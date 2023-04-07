package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/steemit/steemutil/jsonrpc2"
	"github.com/steemit/steemutil/protocol/api"

	"github.com/pkg/errors"
)

type Client struct {
	Url      string
	MaxRetry int
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
