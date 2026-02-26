package core

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"sync"
	"time"
	"os"
)

type TransactionsQueue chan *Transaction
type BlocksQueue chan Block

type Blockchain struct {
	CurrentBlock Block
	BlockSlice

	TransactionsQueue
	BlocksQueue
}

var beginTime map[string]time.Time
var validTxQueue chan *Transaction
var Wg sync.WaitGroup

// Reporter holds running metrics accessible across the package
var Reporter = struct {
	TotalBlocks int
	TotalTxs    int
	TotalTime   float64
}{
	TotalBlocks: 0,
	TotalTxs:    0,
	TotalTime:   0,
}

func init() {
	beginTime = make(map[string]time.Time)
	validTxQueue = make(chan *Transaction, TXPOOL_SIZE)
	Wg.Add(4)
}

func SetupBlockchan() *Blockchain {

	bl := new(Blockchain)
	bl.TransactionsQueue, bl.BlocksQueue = make(TransactionsQueue, TXPOOL_SIZE), make(BlocksQueue)

	//Read blockchain from file and stuff...

	bl.CurrentBlock = bl.CreateNewBlock()

	return bl
}

func (bl *Blockchain) CreateNewBlock() Block {

	prevBlock := bl.BlockSlice.PreviousBlock()
	prevBlockHash := []byte{}
	if prevBlock != nil {

		prevBlockHash = prevBlock.Hash()
	}

	b := NewBlock(prevBlockHash)
	b.BlockHeader.Origin = Core.Keypair.Public

	return b
}

func (bl *Blockchain) AddBlock(b Block) {
	fmt.Printf("Create a new block, tx number [%d]\n", b.TransactionSlice.Len())

	bl.BlockSlice = append(bl.BlockSlice, b)
}

var cnt = 0

func (bl *Blockchain) Run() {

	CurrentBlock := NewBlock(nil)
	interruptBlockGen := bl.GenerateBlocks()
	for {
		select {
		case tr := <-bl.TransactionsQueue:
			/*
				if bl.CurrentBlock.TransactionSlice.Exists(*tr) {
					continue
				}
			*/
			//Server
			/*
				if !tr.VerifyTransaction(TRANSACTION_POW) {
					fmt.Println("Recieved non valid transaction", tr)
					continue
				}
			*/
			/*
				go func() {
					if !tr.VerifyTransaction(TRANSACTION_POW) {
						fmt.Println("Recieved non valid transaction", tr)
					} else {
						validTxQueue <- tr
					}
				}()
			*/
			go func() {
				defer Wg.Done()
				for {
					if !tr.VerifyTransaction(TRANSACTION_POW) {
						fmt.Println("Recieved non valid transaction", tr)
					} else {
						validTxQueue <- tr
					}
				}
			}()

			//cnt++
		//fmt.Printf("Valid Transaction, index [%d]\n", cnt)

		/*
		   Server should generate blocks, the condition has two
		   1. received txs reached BLOCK_TX_NUM
		   2. timeout 3s, if tx not enough, generate a empty or not full blocks
		   Code as follows(two Parts)
		*/
		/*
			//Part I-------
			CurrentBlock.AddTransaction(tr)
			if cnt >= BLOCK_TX_NUM {
				interruptBlockGen <- CurrentBlock
				slice := make(TransactionSlice, BLOCK_TX_NUM)
				CurrentBlock.TransactionSlice = &slice
				cnt = 0
			}-------
		*/
		//interruptBlockGen <- bl.CurrentBlock

		////Broadcast transaction to the network
		//	time.Sleep(time.Millisecond * 10)
		//	mes := NewMessage(MESSAGE_SEND_TRANSACTION)
		//	mes.Data, _ = tr.MarshalBinary()
		//
		//	//time.Sleep(300 * time.Millisecond)
		//	beginTime[hex.EncodeToString(tr.Hash())] = time.Now()
		//	Core.Network.BroadcastQueue <- *mes

		case tr := <-validTxQueue:

			// Broadcast transaction to peers and record timing
			mes := NewMessage(MESSAGE_SEND_TRANSACTION)
			mes.Data, _ = tr.MarshalBinary()
			beginTime[hex.EncodeToString(tr.Hash())] = time.Now()
			Core.Network.BroadcastQueue <- *mes

			cnt++
			CurrentBlock.AddTransaction(tr)
			if cnt >= BLOCK_TX_NUM {

				interruptBlockGen <- CurrentBlock
				slice := make(TransactionSlice, BLOCK_TX_NUM, BLOCK_TX_NUM)
				CurrentBlock.TransactionSlice = &slice
				cnt = 0
			}
			//Part II ------
		case <-time.After(time.Second * BLOCK_GEN_TIMEOUT):
			interruptBlockGen <- CurrentBlock
			slice := make(TransactionSlice, BLOCK_TX_NUM)
			CurrentBlock.TransactionSlice = &slice
			cnt = 0

		case b := <-bl.BlocksQueue:
			_ = b
			/*
				if bl.BlockSlice.Exists(b) {
					fmt.Println("block exists")
					continue
				}
				if !b.VerifyBlock(BLOCK_POW) {
					fmt.Println("block verification fails")
					continue
				}

				if reflect.DeepEqual(b.PrevBlock, bl.CurrentBlock.Hash()) {
					// I'm missing some blocks in the middle. Request'em.
					fmt.Println("Missing blocks in between")
				} else {

					fmt.Println("New block!", b.Hash())

					transDiff := TransactionSlice{}

					if !reflect.DeepEqual(b.BlockHeader.MerkelRoot, bl.CurrentBlock.MerkelRoot) {
						// Transactions are different
						fmt.Println("Transactions are different. finding diff")
						transDiff = DiffTransactionSlices(*bl.CurrentBlock.TransactionSlice, *b.TransactionSlice)
					}

					bl.AddBlock(b)
			*/
			//Broadcast block and shit
			/*
				mes := NewMessage(MESSAGE_SEND_BLOCK)
				mes.Data, _ = b.MarshalBinary()
				Core.Network.BroadcastQueue <- *mes
			*/
			/*
						//New Block
						bl.CurrentBlock = bl.CreateNewBlock()
						bl.CurrentBlock.TransactionSlice = &transDiff

						interruptBlockGen <- bl.CurrentBlock
				}
			*/
		}
	}
}

func DiffTransactionSlices(a, b TransactionSlice) (diff TransactionSlice) {
	//Assumes transaction arrays are sorted (which maybe is too big of an assumption)
	lastj := 0
	for _, t := range a {
		found := false
		for j := lastj; j < len(b); j++ {
			if reflect.DeepEqual(b[j].Signature, t.Signature) {
				found = true
				lastj = j
				break
			}
		}
		if !found {
			diff = append(diff, t)
		}
	}

	return
}
var total int = 0

func (bl *Blockchain) GenerateBlocks() chan Block {

	interrupt := make(chan Block)

	// Metrics
	var lastBlockTime time.Time

	// Ensure report file exists
	go func() {
		f, _ := os.OpenFile("tps_report.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		defer f.Close()
		f.WriteString("--- TPS Reporter Started at " + time.Now().String() + " ---\n")
	}()

	go func() {
		for {
			block := <-interrupt

			if total == 0{
				total += 1
				continue

			}

			block.BlockHeader.Timestamp = uint32(time.Now().Unix())
			blockHash := hex.EncodeToString(block.Hash())
			fmt.Printf("Generate a Block [%s]\n", blockHash)
			beginTime[blockHash] = time.Now()

			// Per-block TPS calculation
			now := time.Now()
			var delta float64
			if !lastBlockTime.IsZero() {
				delta = now.Sub(lastBlockTime).Seconds()
			} else {
				delta = 0
			}
			lastBlockTime = now

			n := block.TransactionSlice.Len()
	Reporter.TotalBlocks += 1
	Reporter.TotalTxs += n

	var perBlockTPS float64
	if delta > 0 {
		perBlockTPS = float64(n) / delta
	} else {
		perBlockTPS = 0
	}

	// Aggregate
	used := 0.0
	if bt, ok := beginTime[blockHash]; ok {
		used = time.Now().Sub(bt).Seconds()
		Reporter.TotalTime += used
	}

	avgTPS := 0.0
	if Reporter.TotalTime > 0 {
		avgTPS = float64(Reporter.TotalTxs) / Reporter.TotalTime
	}

	// Log to stdout and file
	logLine := fmt.Sprintf("[%s] Block %d: tx=%d, per_block_tps=%.2f, total_tx=%d, avg_tps=%.2f\n", now.Format(time.RFC3339), Reporter.TotalBlocks, n, perBlockTPS, Reporter.TotalTxs, avgTPS)
	fmt.Print(logLine)
	f, err := os.OpenFile("tps_report.log", os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(logLine)
		f.Close()
	}

	print("Send a block contains " , n, " tx\n")
	mes := NewMessage(MESSAGE_SEND_BLOCK)
	mes.Data, _ = block.MarshalBinary()

	Core.Network.BroadcastQueue <- *mes

	time.Sleep(time.Second * BLOCK_BROADCAST_INTERVAL)
	}
	}()

	return interrupt
}

// DumpReport writes current reporter summary to disk
func DumpReport() error {
	f, err := os.OpenFile("tps_report.log", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	avg := 0.0
	if Reporter.TotalTime > 0 {
		avg = float64(Reporter.TotalTxs) / Reporter.TotalTime
	}
	line := fmt.Sprintf("--- Dump at %s: total_blocks=%d total_txs=%d total_time=%.3f avg_tps=%.2f ---\n", time.Now().Format(time.RFC3339), Reporter.TotalBlocks, Reporter.TotalTxs, Reporter.TotalTime, avg)
	_, err = f.WriteString(line)
	return err
}
