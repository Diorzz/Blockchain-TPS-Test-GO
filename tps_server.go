package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type BlockEntry struct {
	Timestamp   string  `json:"timestamp"`
	Block       int     `json:"block"`
	Tx          int     `json:"tx"`
	PerBlockTPS float64 `json:"per_block_tps"`
}

type Metrics struct {
	TotalBlocks int          `json:"total_blocks"`
	TotalTxs    int          `json:"total_txs"`
	TotalTime   float64      `json:"total_time"`
	AvgTPS      float64      `json:"avg_tps"`
	LastBlocks  []BlockEntry `json:"last_blocks"`
}

func parseReport(path string) (*Metrics, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(b), "\n")
	var last []string
	for i := len(lines) - 1; i >= 0 && len(last) < 50; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = append([]string{lines[i]}, last...)
		}
	}

	m := &Metrics{}
	// Simple parse: look for Dump line or block lines
	reBlock := regexp.MustCompile(`\[(.+)\] Block (\d+): tx=(\d+):?, per_block_tps=([0-9.]+), total_tx=(\d+), avg_tps=([0-9.]+)`)
	reDump := regexp.MustCompile(`--- Dump at (.+): total_blocks=(\d+) total_txs=(\d+) total_time=([0-9.]+) avg_tps=([0-9.]+) ---`)

	for _, l := range last {
		if m2 := reBlock.FindStringSubmatch(l); m2 != nil {
			blockNum, _ := strconv.Atoi(m2[2])
			txNum, _ := strconv.Atoi(m2[3])
			per, _ := strconv.ParseFloat(m2[4], 64)
			m.LastBlocks = append(m.LastBlocks, BlockEntry{Timestamp: m2[1], Block: blockNum, Tx: txNum, PerBlockTPS: per})
		}
		if m3 := reDump.FindStringSubmatch(l); m3 != nil {
			m.TotalBlocks, _ = strconv.Atoi(m3[2])
			m.TotalTxs, _ = strconv.Atoi(m3[3])
			m.TotalTime, _ = strconv.ParseFloat(m3[4], 64)
			m.AvgTPS, _ = strconv.ParseFloat(m3[5], 64)
		}
	}

	// fallback: attempt to find avg_tps in last block lines
	if m.AvgTPS == 0 && len(m.LastBlocks) > 0 {
		// average last N per-block tps
		sum := 0.0
		count := 0
		for _, be := range m.LastBlocks {
			sum += be.PerBlockTPS
			count++
		}
		if count > 0 {
			m.AvgTPS = sum / float64(count)
		}
	}

	return m, nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	m, err := parseReport("tps_report.log")
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading report: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func main() {
	http.HandleFunc("/metrics", metricsHandler)

	// Serve static files from txps/dist (after build)
	dist := "txps/dist"
	if _, err := os.Stat(dist); os.IsNotExist(err) {
		// try public
		if _, err2 := os.Stat("txps/public"); err2 == nil {
			dist = "txps/public"
		}
	}

	fs := http.FileServer(http.Dir(dist))
	http.Handle("/", fs)

	port := "8081"
	log.Printf("Starting TPS server on http://localhost:%s (static=%s)", port, dist)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
