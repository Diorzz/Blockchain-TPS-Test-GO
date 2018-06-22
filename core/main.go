package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

var Core = struct {
	*Keypair
	*Blockchain
	*Network
}{}

func Start(address string) {

	// Setup keys
	keypair, _ := OpenConfiguration(HOME_DIRECTORY_CONFIG)
	if keypair == nil {

		fmt.Println("Generating keypair...")
		keypair = GenerateNewKeypair()
		WriteConfiguration(HOME_DIRECTORY_CONFIG, keypair)
	}
	Core.Keypair = keypair

	// Setup Network
	Core.Network = SetupNetwork(address, BLOCKCHAIN_PORT)
	go Core.Network.Run()
	for _, n := range SEED_NODES() {
		Core.Network.ConnectionsQueue <- n
	}

	// Setup blockchain
	Core.Blockchain = SetupBlockchan()
	go Core.Blockchain.Run()

	go func() {
		for {
			select {
			case msg := <-Core.Network.IncomingMessages:
				HandleIncomingMessage(msg)
			}
		}
	}()
}

func CreateTransaction(txt string) *Transaction {

	t := NewTransaction(Core.Keypair.Public, nil, []byte(txt))
	t.Header.Nonce = t.GenerateNonce(TRANSACTION_POW)
	t.Signature = t.Sign(Core.Keypair)

	return t
}

//var cnt = 0
var count int = 0
var totalTime float64 = 0
var SEND int = 10000
func HandleIncomingMessage(msg Message) {


	switch msg.Identifier {
	case MESSAGE_SEND_TRANSACTION:
		t := new(Transaction)
		_, err := t.UnmarshalBinary(msg.Data)
		if err != nil {
			networkError(err)
			break
		}
		count += 1

		trHax := hex.EncodeToString(t.Hash())
		du := time.Now().Sub(beginTime[trHax]).Seconds()
		totalTime += du
		if count >= SEND {

			fmt.Println("Receive ", SEND, " valid tx, total time is :", totalTime)
			panic("xxxxx")
		}


		//Core.Blockchain.TransactionsQueue <- t

	case MESSAGE_SEND_BLOCK:
		b := new(Block)
		err := b.UnmarshalBinary(msg.Data)
		if err != nil {
			networkError(err)
			break
		}
		print("Receive a block contains ", b.TransactionSlice.Len(), " tx\n")
		blockHash := hex.EncodeToString(b.Hash())
		fmt.Printf("Recieve a block [%s]\n", blockHash)
		//if value, ok := beginTime[blockHash]; ok {
		usedTime := time.Now().Sub(beginTime[blockHash]).Seconds()
		txsNumber := BLOCK_TX_NUM
		fmt.Printf("Tx_num: %d, usedTime: %fs, tps: %f\n", txsNumber, usedTime, float64(txsNumber)/usedTime)
		//}
		//Core.Blockchain.BlocksQueue <- *b
	}
}

func logOnError(err error) {

	if err != nil {
		log.Println("[Todos] Err:", err)
	}
}
