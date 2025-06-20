package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type LoadTestConfig struct {
	TargetURL       string
	ConcurrentUsers int
	Duration        time.Duration
	RequestsPerSec  int
}

type SensorData struct {
	SensorID   string                 `json:"sensor_id"`
	Timestamp  time.Time              `json:"timestamp"`
	MetricType string                 `json:"metric_type"`
	Value      float64                `json:"value"`
	Tags       map[string]string      `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type BulkSensorData struct {
	Data []SensorData `json:"data"`
}

type TestResults struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	Errors          []string
	mu              sync.RWMutex
}

func (tr *TestResults) AddResult(success bool, latency time.Duration, err error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.TotalRequests++
	tr.TotalLatency += latency

	if tr.MinLatency == 0 || latency < tr.MinLatency {
		tr.MinLatency = latency
	}
	if latency > tr.MaxLatency {
		tr.MaxLatency = latency
	}

	if success {
		tr.SuccessRequests++
	} else {
		tr.FailedRequests++
		if err != nil {
			tr.Errors = append(tr.Errors, err.Error())
		}
	}
}

func (tr *TestResults) GetStats() (float64, float64, time.Duration) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	successRate := float64(tr.SuccessRequests) / float64(tr.TotalRequests) * 100
	avgLatency := tr.TotalLatency / time.Duration(tr.TotalRequests)

	return successRate, float64(tr.TotalRequests), avgLatency
}

func generateSensorData(count int) BulkSensorData {
	sensorTypes := []string{"temperature", "humidity", "pressure", "light", "motion", "sound"}
	sensorIDs := []string{"sensor_001", "sensor_002", "sensor_003", "sensor_004", "sensor_005", "sensor_006", "sensor_007", "sensor_008", "sensor_009", "sensor_010"}
	locations := []string{"floor1", "floor2", "floor3", "basement", "roof"}

	data := make([]SensorData, count)

	for i := 0; i < count; i++ {
		sensorType := sensorTypes[rand.Intn(len(sensorTypes))]
		sensorID := sensorIDs[rand.Intn(len(sensorIDs))]
		location := locations[rand.Intn(len(locations))]

		var value float64
		switch sensorType {
		case "temperature":
			value = rand.Float64()*50 + 10 // 10-60°C
		case "humidity":
			value = rand.Float64() * 100 // 0-100%
		case "pressure":
			value = rand.Float64()*200 + 900 // 900-1100 hPa
		case "light":
			value = rand.Float64() * 1000 // 0-1000 lux
		case "motion":
			value = float64(rand.Intn(2)) // 0 or 1
		case "sound":
			value = rand.Float64()*80 + 20 // 20-100 dB
		}

		data[i] = SensorData{
			SensorID:   sensorID,
			Timestamp:  time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second),
			MetricType: sensorType,
			Value:      value,
			Tags: map[string]string{
				"location": location,
				"building": "main",
				"floor":    fmt.Sprintf("floor_%d", rand.Intn(5)+1),
			},
			Metadata: map[string]interface{}{
				"device_id":       fmt.Sprintf("device_%d", rand.Intn(100)),
				"battery_level":   rand.Float64() * 100,
				"signal_strength": rand.Intn(100),
			},
		}
	}

	return BulkSensorData{Data: data}
}

func sendRequest(client *http.Client, url string, data BulkSensorData) (bool, time.Duration, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, 0, err
	}

	start := time.Now()

	req, err := http.NewRequest("POST", url+"/api/v1/ingest", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, time.Since(start), err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, time.Since(start), err
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	if !success {
		return false, latency, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return true, latency, nil
}

func worker(ctx context.Context, workerID int, config LoadTestConfig, results *TestResults, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	ticker := time.NewTicker(time.Second / time.Duration(config.RequestsPerSec))
	defer ticker.Stop()

	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopped", workerID)
			return
		case <-ticker.C:
			// Generate test data with varying batch sizes
			batchSize := rand.Intn(50) + 10 // 10-60 data points per batch
			testData := generateSensorData(batchSize)

			success, latency, err := sendRequest(client, config.TargetURL, testData)
			results.AddResult(success, latency, err)
		}
	}
}

func printProgress(ctx context.Context, results *TestResults, duration time.Duration) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(start)
			remaining := duration - elapsed

			successRate, totalReqs, avgLatency := results.GetStats()

			fmt.Printf("\n=== Progress Update ===\n")
			fmt.Printf("Elapsed: %v, Remaining: %v\n", elapsed.Round(time.Second), remaining.Round(time.Second))
			fmt.Printf("Total Requests: %.0f\n", totalReqs)
			fmt.Printf("Success Rate: %.2f%%\n", successRate)
			fmt.Printf("Average Latency: %v\n", avgLatency.Round(time.Millisecond))
			fmt.Printf("Requests/sec: %.2f\n", totalReqs/elapsed.Seconds())

			if remaining <= 0 {
				return
			}
		}
	}
}

func testAggregateEndpoint(client *http.Client, baseURL string) error {
	fmt.Println("\n=== Testing Aggregate Endpoint ===")

	query := map[string]interface{}{
		"start_time":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":    time.Now().Format(time.RFC3339),
		"granularity": "minute",
	}

	jsonData, _ := json.Marshal(query)

	start := time.Now()
	resp, err := client.Post(baseURL+"/api/v1/aggregates", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("aggregate request failed: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode != 200 {
		return fmt.Errorf("aggregate endpoint returned HTTP %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode aggregate response: %w", err)
	}

	fmt.Printf("Aggregate query completed in %v\n", latency.Round(time.Millisecond))
	fmt.Printf("Results count: %v\n", result["count"])

	return nil
}

func main() {
	config := LoadTestConfig{
		TargetURL:       getEnv("TARGET_URL", "http://localhost:8080"),
		ConcurrentUsers: getEnvInt("CONCURRENT_USERS", 10),
		Duration:        getEnvDuration("DURATION", "60s"),
		RequestsPerSec:  getEnvInt("REQUESTS_PER_SEC", 5),
	}

	fmt.Printf("=== Load Test Configuration ===\n")
	fmt.Printf("Target URL: %s\n", config.TargetURL)
	fmt.Printf("Concurrent Users: %d\n", config.ConcurrentUsers)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Requests per second per user: %d\n", config.RequestsPerSec)
	fmt.Printf("Total expected requests per second: %d\n", config.ConcurrentUsers*config.RequestsPerSec)

	// Wait for service to be ready
	fmt.Println("\nWaiting for service to be ready...")
	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < 30; i++ {
		resp, err := client.Get(config.TargetURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			fmt.Println("Service is ready!")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}

		fmt.Printf("Waiting for service... (%d/30)\n", i+1)
		time.Sleep(2 * time.Second)
	}

	// Initialize results tracking
	results := &TestResults{}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Start progress reporting
	go printProgress(ctx, results, config.Duration)

	// Start workers
	var wg sync.WaitGroup
	fmt.Printf("\nStarting %d concurrent users...\n", config.ConcurrentUsers)

	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go worker(ctx, i+1, config, results, &wg)
	}

	// Wait for test completion
	wg.Wait()

	// Final results
	fmt.Printf("\n=== Final Results ===\n")
	successRate, totalReqs, avgLatency := results.GetStats()

	fmt.Printf("Total Requests: %.0f\n", totalReqs)
	fmt.Printf("Successful Requests: %d\n", results.SuccessRequests)
	fmt.Printf("Failed Requests: %d\n", results.FailedRequests)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Average Latency: %v\n", avgLatency.Round(time.Millisecond))
	fmt.Printf("Min Latency: %v\n", results.MinLatency.Round(time.Millisecond))
	fmt.Printf("Max Latency: %v\n", results.MaxLatency.Round(time.Millisecond))
	fmt.Printf("Throughput: %.2f requests/second\n", totalReqs/config.Duration.Seconds())

	if len(results.Errors) > 0 {
		fmt.Printf("\n=== Errors (showing first 10) ===\n")
		for i, err := range results.Errors {
			if i >= 10 {
				fmt.Printf("... and %d more errors\n", len(results.Errors)-10)
				break
			}
			fmt.Printf("- %s\n", err)
		}
	}

	// Test aggregate endpoint
	if err := testAggregateEndpoint(&http.Client{Timeout: 30 * time.Second}, config.TargetURL); err != nil {
		fmt.Printf("Aggregate endpoint test failed: %v\n", err)
	}

	fmt.Println("\nLoad test completed!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	if parsed, err := time.ParseDuration(defaultValue); err == nil {
		return parsed
	}
	return time.Minute
}
