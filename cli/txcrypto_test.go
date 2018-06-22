package main

import (
	"testing"

	"github.com/izqui/blockchain/core"
)

func TestCreateTransactionTest(t *testing.T) {
	tx := CreateTransactionTest("11")
	if !tx.VerifyTransaction(core.TRANSACTION_POW) {
		t.Fatal("verify failed")
	}
}
