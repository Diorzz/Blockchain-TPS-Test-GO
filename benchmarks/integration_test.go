package main

import (
	"testing"

	"tps-testing/core"
)

/*
func Test_SendTxs(t *testing.T) {
	core.Core.Blockchain.TransactionsQueue <- core.CreateTransaction("hello world")
}

//Small tx Test, 8byte
func Benchmark_Txs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		core.Core.Blockchain.TransactionsQueue <- core.CreateTransaction("1234abcd")
	}
}

//Big tx Test
//Benchmark1k  Benchmark4k Benchmark16k Benchmark64k Benchmark128k Benchmark512k Benchmark1M Benchmark4M Benchmark16M
//the suffix is to control the size of a tx
func benchmark(b *testing.B, size int) {
	tx := make([]byte, size*1024)
	for i := 0; i < size; i++ {
		tx[i] = 'a'
	}
	for i := 0; i < b.N; i++ {
		core.Core.Blockchain.TransactionsQueue <- core.CreateTransaction(strconv.Itoa(i + 10))
	}
}
func Benchmark1k(b *testing.B) {
	benchmark(b, 1)
}
func Benchmark64k(b *testing.B) {
	benchmark(b, 64)
}

func Benchmark1M(b *testing.B) {
	benchmark(b, 1*1024)
}
*/
func BenchmarkTxSize(b *testing.B) {
	core.Start("127.0.0.1:8888")
	b.ResetTimer()
	//testCases := []string{"80", "200", "512", strconv.Itoa(1 * 1024), strconv.Itoa(4 * 1024), strconv.Itoa(16 * 1024)} //80b -> 16k
	testCases := []string{"80"} //80b -> 16k

	for _, txSize := range testCases {
		b.Run(txSize, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				core.Core.Blockchain.TransactionsQueue <- core.CreateTransaction(txSize) // Deferred to mempool
			}
		})
	}
}
