package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/steemit/steemgosdk/consts"
	"github.com/steemit/steemutil/jsonrpc2"
	"github.com/steemit/steemutil/protocol"
	"github.com/steemit/steemutil/protocol/api"
	"github.com/steemit/steemutil/transaction"
	"github.com/steemit/steemutil/wif"

	"github.com/pkg/errors"
)

type Client struct {
	Url         string
	MaxRetry    int
	AccountName string
	Wifs        map[string]*wif.PrivateKey
}

type WrapBlock struct {
	BlockNum uint
	Block    *api.Block
}

func (c *Client) GetRpcClient() *jsonrpc2.JsonRpc {
	return jsonrpc2.NewClient(c.Url)
}

func (c *Client) GetDynamicGlobalProperties() (dgp *api.DynamicGlobalProperties, err error) {
	rpc := c.GetRpcClient()
	err = rpc.BuildSendData(
		"condenser_api.get_dynamic_global_properties",
		[]any{},
	)
	if err != nil {
		return
	}
	rpcResponse, err := rpc.Send()
	if err != nil {
		return
	}
	if rpcResponse.Error != nil {
		return dgp, errors.Errorf("failed to GetDynamicGlobalProperties:%v", rpcResponse.Error)
	}
	tmp, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return
	}
	dgp = &api.DynamicGlobalProperties{}
	json.Unmarshal(tmp, dgp)
	return
}

func (c *Client) GetBlock(blockNum uint) (block *api.Block, err error) {
	rpc := c.GetRpcClient()
	err = rpc.BuildSendData(
		"condenser_api.get_block",
		[]any{blockNum},
	)
	if err != nil {
		return
	}
	rpcResponse, err := rpc.Send()
	if err != nil {
		return
	}
	if rpcResponse.Error != nil {
		return block, errors.Errorf("failed to GetBlock:%v", rpcResponse.Error)
	}
	tmp, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return
	}
	block = &api.Block{}
	json.Unmarshal(tmp, block)
	return
}

func (c *Client) BroadcastSync(params []any) (resultJson []byte, err error) {
	rpc := c.GetRpcClient()
	err = rpc.BuildSendData(
		"condenser_api.broadcast_transaction_synchronous",
		params,
	)
	if err != nil {
		return
	}
	fmt.Printf("test send data: %+v", string(rpc.SendData))
	rpcResponse, err := rpc.Send()
	if err != nil {
		return
	}
	if rpcResponse.Error != nil {
		return resultJson, errors.Errorf("failed to broadcast:%v", rpcResponse.Error)
	}
	resultJson, err = json.Marshal(rpcResponse.Result)
	return
}

func (c *Client) wrapGetBlock(blockNum uint, ch chan<- *WrapBlock) {
	var (
		err   error
		block *api.Block
	)

	for i := 0; i < c.MaxRetry; i++ {
		block, err = c.GetBlock(blockNum)
		if err == nil {
			break
		}
		fmt.Printf("retry get block {%+v} after 1 second.", blockNum)
		time.Sleep(time.Second)
	}
	if err != nil {
		fmt.Printf("wrapGetBlock err: %+v", err)
		ch <- nil
		return
	}
	ch <- &WrapBlock{
		BlockNum: blockNum,
		Block:    block,
	}
}

// get_blocks [from, to)
func (c *Client) GetBlocks(from, to uint) (blocks []*WrapBlock, err error) {
	// check params
	if from >= to {
		return blocks, errors.Errorf("unexpected params {from: %v}, {to: %v}", from, to)
	}
	// init
	ch := make(chan *WrapBlock, to-from)
	blocksMap := make(map[uint]*WrapBlock, to-from)
	blocks = make([]*WrapBlock, 0, to-from)
	// get blocks
	for i := from; i < to; i++ {
		go c.wrapGetBlock(i, ch)
	}
	// get results
	for i := from; i < to; i++ {
		result := <-ch
		blocksMap[result.BlockNum] = result
		if blocksMap[result.BlockNum] == nil {
			return blocks, errors.Errorf("get block {%v} error", result.BlockNum)
		}
	}
	// sort result
	for i := from; i < to; i++ {
		blocks = append(blocks, blocksMap[i])
	}
	return
}

func (c *Client) ImportWif(keyType string, privWif string) (err error) {
	if !checkKeyType(keyType) {
		return errors.New("unexpected keyType when import wif")
	}
	priv := &wif.PrivateKey{}
	err = priv.FromWif(privWif)
	if err != nil {
		return
	}
	if len(c.Wifs) == 0 {
		c.Wifs = make(map[string]*wif.PrivateKey, 0)
	}
	c.Wifs[keyType] = priv
	return
}

func (c *Client) GetTransactionHex(tx *transaction.SignedTransaction) (result any, err error) {
	rpc := c.GetRpcClient()
	err = rpc.BuildSendData(
		"condenser_api.get_transaction_hex",
		[]any{tx.Transaction},
	)
	if err != nil {
		return
	}
	rpcResponse, err := rpc.Send()
	if err != nil {
		return
	}
	if rpcResponse.Error != nil {
		return result, errors.Errorf("failed to GetTransactionHex:%v", rpcResponse.Error)
	}
	result = rpcResponse.Result
	return
}

func (c *Client) BroadcastRawOps(ops []protocol.Operation, priv *wif.PrivateKey) (err error) {
	if len(ops) == 0 {
		return errors.Errorf("no operations submit")
	}

	dgp, err := c.GetDynamicGlobalProperties()
	if err != nil {
		return err
	}

	// Prepare the transaction.
	refBlockPrefix, err := transaction.RefBlockPrefix(dgp.HeadBlockId)
	if err != nil {
		return err
	}

	tx := transaction.NewSignedTransaction(&transaction.Transaction{
		RefBlockNum:    transaction.RefBlockNum(dgp.HeadBlockNumber),
		RefBlockPrefix: refBlockPrefix,
	})

	for _, op := range ops {
		tx.PushOperation(op)
	}

	err = tx.Sign([]*wif.PrivateKey{priv}, transaction.SteemChain)
	if err != nil {
		return err
	}
	if len(tx.Signatures) != 1 {
		return errors.Errorf("expected signatures not appended to the transaction")
	}
	_, err = c.BroadcastSync([]any{tx})
	return
}

func checkKeyType(keyType string) (r bool) {
	if keyType == consts.ACTIVE_KEY {
		return true
	}
	if keyType == consts.POSTING_KEY {
		return true
	}
	if keyType == consts.OWNER_KEY {
		return true
	}
	if keyType == consts.MEMO_KEY {
		return true
	}
	return false
}
