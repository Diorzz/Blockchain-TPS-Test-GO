package main

// Go TPS Tester → Inferstructor GPU Bridge Adapter
// 
// This adapter bridges the existing Blockchain-TPS-Test-GO tool
// with the Inferstructor GPU acceleration lanes.
//
// Usage:
//   go run tps_inferstructor_adapter.go --target-tps 19500000 --duration 600

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// AccelerationRequest matches Python bridge API
type AccelerationRequest struct {
	TxHash string `json:"tx_hash"`
	TxData string `json:"tx_data"` // hex encoded
	Chain  string `json:"chain"`
}

// AccelerationResponse from GPU lane
type AccelerationResponse struct {
	Success    bool    `json:"success"`
	TxHash     string  `json:"tx_hash"`
	Result     string  `json:"result"`
	ResultHash string  `json:"result_hash"`
	LaneID     string  `json:"lane_id"`
	LatencyMs  float64 `json:"latency_ms"`
	Error      string  `json:"error,omitempty"`
}

// BatchRequest for higher throughput
type BatchRequest struct {
	Transactions []AccelerationRequest `json:"transactions"`
}

// BatchResponse from bridge
type BatchResponse struct {
	Results []AccelerationResponse `json:"results"`
}

// Stats tracking
type Stats struct {
	TotalSent      uint64
	TotalSucceeded uint64
	TotalFailed    uint64
	TotalLatencyMs uint64
	StartTime      time.Time
}

var (
	bridgeEndpoint = flag.String("bridge", "http://localhost:9999", "TPS Bridge endpoint")
	targetTPS      = flag.Int("target-tps", 19500000, "Target transactions per second")
	duration       = flag.Int("duration", 600, "Test duration in seconds")
	batchSize      = flag.Int("batch-size", 1000, "Batch size for requests")
	workers        = flag.Int("workers", 1000, "Number of concurrent workers")
	chain          = flag.String("chain", "generic", "Chain identifier")
	verbose        = flag.Bool("verbose", false, "Verbose logging")
)

var stats Stats

func main() {
	flag.Parse()

	log.Printf("Starting Inferstructor TPS Test")
	log.Printf("  Bridge: %s", *bridgeEndpoint)
	log.Printf("  Target TPS: %d", *targetTPS)
	log.Printf("  Duration: %ds", *duration)
	log.Printf("  Batch Size: %d", *batchSize)
	log.Printf("  Workers: %d", *workers)
	log.Printf("  Chain: %s", *chain)

	stats.StartTime = time.Now()

	// Check bridge health
	if !checkBridgeHealth() {
		log.Fatal("Bridge health check failed")
	}

	// Calculate rate per worker
	txPerSecond := *targetTPS
	txPerWorker := txPerSecond / *workers
	if txPerWorker == 0 {
		txPerWorker = 1
	}

	log.Printf("Each worker will send ~%d tx/s", txPerWorker)

	// Start stats reporter
	go statsReporter()

	// Launch workers
	var wg sync.WaitGroup
	endTime := time.Now().Add(time.Duration(*duration) * time.Second)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(i, txPerWorker, endTime, &wg)
	}

	log.Println("Workers started. Generating load...")

	wg.Wait()

	// Final stats
	elapsed := time.Since(stats.StartTime).Seconds()
	totalSent := atomic.LoadUint64(&stats.TotalSent)
	totalSucceeded := atomic.LoadUint64(&stats.TotalSucceeded)
	totalFailed := atomic.LoadUint64(&stats.TotalFailed)
	totalLatency := atomic.LoadUint64(&stats.TotalLatencyMs)

	avgLatency := 0.0
	if totalSucceeded > 0 {
		avgLatency = float64(totalLatency) / float64(totalSucceeded)
	}

	actualTPS := float64(totalSucceeded) / elapsed

	log.Println("\n=== Final Results ===")
	log.Printf("Duration: %.2fs", elapsed)
	log.Printf("Total Sent: %d", totalSent)
	log.Printf("Total Succeeded: %d", totalSucceeded)
	log.Printf("Total Failed: %d", totalFailed)
	log.Printf("Success Rate: %.2f%%", float64(totalSucceeded)/float64(totalSent)*100)
	log.Printf("Actual TPS: %.2f", actualTPS)
	log.Printf("Target TPS: %d", *targetTPS)
	log.Printf("Achieved: %.2f%% of target", actualTPS/float64(*targetTPS)*100)
	log.Printf("Avg Latency: %.3fms", avgLatency)

	// Compare to Solana baseline (65K TPS)
	solanaBaseline := 65000.0
	speedup := actualTPS / solanaBaseline
	log.Printf("\nSpeedup vs Solana: %.2fx", speedup)

	if speedup >= 300 {
		log.Println("✅ 300× Solana speed achieved!")
	} else {
		log.Printf("⚠️  Target 300× not reached (%.2fx achieved)", speedup)
	}
}

func worker(id int, txPerSecond int, endTime time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	// Calculate delay between batches
	batchesPerSecond := float64(txPerSecond) / float64(*batchSize)
	delayBetweenBatches := time.Duration(1000/batchesPerSecond) * time.Millisecond

	if *verbose {
		log.Printf("Worker %d: %d tx/s, batch delay: %v", id, txPerSecond, delayBetweenBatches)
	}

	ticker := time.NewTicker(delayBetweenBatches)
	defer ticker.Stop()

	for time.Now().Before(endTime) {
		<-ticker.C

		// Generate and send batch
		batch := generateBatch(*batchSize, id)
		succeeded, failed, avgLatency := sendBatch(client, batch)

		atomic.AddUint64(&stats.TotalSent, uint64(*batchSize))
		atomic.AddUint64(&stats.TotalSucceeded, uint64(succeeded))
		atomic.AddUint64(&stats.TotalFailed, uint64(failed))
		if succeeded > 0 {
			atomic.AddUint64(&stats.TotalLatencyMs, uint64(avgLatency*float64(succeeded)))
		}
	}

	if *verbose {
		log.Printf("Worker %d finished", id)
	}
}

func generateBatch(size int, workerId int) []AccelerationRequest {
	batch := make([]AccelerationRequest, size)

	for i := 0; i < size; i++ {
		// Generate synthetic transaction data
		txData := make([]byte, 128)
		// Simple pattern: worker ID + sequence
		copy(txData, []byte(fmt.Sprintf("worker%d-tx%d", workerId, i)))

		txHash := fmt.Sprintf("%064x", time.Now().UnixNano()+int64(i))

		batch[i] = AccelerationRequest{
			TxHash: txHash,
			TxData: hex.EncodeToString(txData),
			Chain:  *chain,
		}
	}

	return batch
}

func sendBatch(client *http.Client, batch []AccelerationRequest) (succeeded int, failed int, avgLatency float64) {
	batchReq := BatchRequest{
		Transactions: batch,
	}

	jsonData, err := json.Marshal(batchReq)
	if err != nil {
		log.Printf("Error marshaling batch: %v", err)
		return 0, len(batch), 0
	}

	start := time.Now()

	resp, err := client.Post(
		*bridgeEndpoint+"/accelerate/batch",
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		if *verbose {
			log.Printf("Error sending batch: %v", err)
		}
		return 0, len(batch), 0
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()

	if resp.StatusCode != 200 {
		if *verbose {
			log.Printf("Bad response: %d", resp.StatusCode)
		}
		return 0, len(batch), 0
	}

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		if *verbose {
			log.Printf("Error decoding response: %v", err)
		}
		return 0, len(batch), 0
	}

	// Count successes
	for _, result := range batchResp.Results {
		if result.Success {
			succeeded++
		} else {
			failed++
		}
	}

	if succeeded > 0 {
		avgLatency = float64(latency) / float64(succeeded)
	}

	return succeeded, failed, avgLatency
}

func checkBridgeHealth() bool {
	log.Println("Checking bridge health...")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(*bridgeEndpoint + "/health")
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Health check returned %d", resp.StatusCode)
		return false
	}

	log.Println("✓ Bridge is healthy")
	return true
}

func statsReporter() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		elapsed := time.Since(stats.StartTime).Seconds()
		if elapsed == 0 {
			continue
		}

		sent := atomic.LoadUint64(&stats.TotalSent)
		succeeded := atomic.LoadUint64(&stats.TotalSucceeded)
		failed := atomic.LoadUint64(&stats.TotalFailed)

		currentTPS := float64(succeeded) / elapsed

		log.Printf("[Stats] Sent: %d | Succeeded: %d | Failed: %d | TPS: %.2f",
			sent, succeeded, failed, currentTPS)
	}
}
