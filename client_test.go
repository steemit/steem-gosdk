package main

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/steemit/steemutil/protocol"
	"github.com/steemit/steemutil/transaction"
)

var (
	wifs = map[string]string{
		"memo":    "5KhRtadrBoa9MEke7AoH2z73qH6GJGKRzUnXVVkaprgcqKrXwwX",
		"active":  "5KjrKfLLRkDnY8cHYH2PkMofv6W4xwykatdqyUgQ7eCHDwkjAwf",
		"posting": "5K4YjdpHFUJpMoWV7u1KTnAaZy59N8oT4csQdwyyqhLqCyZZQ6U",
		"owner":   "5KVCK2NsPxQVHLcjD2FCNtGyp7agdrv8nn1WAn9HnDybNLBhsEm",
	}
)

func TestImportWif(t *testing.T) {
	client := &Client{
		Url:      "https://api.steemit.com",
		MaxRetry: 5,
	}

	for kType, wif := range wifs {
		err := client.ImportWif(kType, wif)
		if err != nil {
			t.Errorf("testImportWif error: %+v", err)
			return
		}
	}

	for kType, wif := range wifs {
		if client.Wifs[kType].ToWif() != wif {
			t.Errorf("TestImportWif unexpect wif, kType: %+v, expected: %+v, got: %+v", kType, wif, client.Wifs[kType].ToWif())
			return
		}
	}
}

func TestBroadcast(t *testing.T) {
	client := &Client{
		Url:      "https://api.steemit.com",
		MaxRetry: 5,
	}

	username := "ety001.test"
	// TODO: need a mock
	client.ImportWif("posting", "")

	comment := &protocol.CommentOperation{
		Author:         username,
		Title:          "test from go sdk",
		Permlink:       "test-from-go-sdk",
		ParentAuthor:   "",
		ParentPermlink: "test",
		Body:           "test from go sdk content",
	}

	err := client.BroadcastRawOps([]protocol.Operation{comment}, client.Wifs["posting"])
	if err != nil {
		t.Errorf("broadcast error: %v", err)
	}

}

func TestGetTransactionHex(t *testing.T) {
	client := &Client{
		Url:      "https://api.steemit.com",
		MaxRetry: 5,
	}
	username := "ety001"
	comment := &protocol.CommentOperation{
		Author:         username,
		Title:          "test post",
		Permlink:       "ety001-test-post",
		ParentAuthor:   "",
		ParentPermlink: "test",
		Body:           "test post body",
		JsonMetadata:   "{}",
	}
	dgp, err := client.GetDynamicGlobalProperties()
	if err != nil {
		t.Errorf("dgp error: %+v", err)
	}
	// Prepare the transaction.
	refBlockPrefix, err := transaction.RefBlockPrefix(dgp.HeadBlockId)
	if err != nil {
		t.Errorf("ref error: %+v", err)
	}
	expiration := time.Date(2016, 8, 8, 12, 24, 17, 0, time.UTC)
	tx := transaction.NewSignedTransaction(&transaction.Transaction{
		RefBlockNum:    transaction.RefBlockNum(dgp.HeadBlockNumber),
		RefBlockPrefix: refBlockPrefix,
		Expiration:     &protocol.Time{Time: &expiration},
		Signatures:     []string{},
	})
	tx.PushOperation(comment)

	txBytes, err := tx.Serialize()
	if err != nil {
		t.Errorf("GetTransactionHex error: %+v", err)
	}
	got := hex.EncodeToString(txBytes)

	result, err := client.GetTransactionHex(tx)
	if err != nil {
		t.Errorf("GetTransactionHex error: %+v", err)
	}

	if got+"00" != fmt.Sprintf("%v", result) {
		t.Errorf("expected: %+v, got: %+v", fmt.Sprintf("%v", result), got+"00")
	}
}
