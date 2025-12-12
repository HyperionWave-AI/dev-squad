package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RequestLog represents a logged AI request
type RequestLog struct {
	Timestamp   string                 `json:"timestamp"`
	Provider    string                 `json:"provider"`
	MsgContents []interface{}          `json:"msgContents"`
	Tools       []interface{}          `json:"tools"`
	Options     map[string]interface{} `json:"options"`
}

// ResponseLog represents a logged AI response
type ResponseLog struct {
	Timestamp string                 `json:"timestamp"`
	Provider  string                 `json:"provider"`
	Response  map[string]interface{} `json:"response"`
	Error     string                 `json:"error,omitempty"`
}

func main() {
	logsDir := "./logs"

	if len(os.Args) > 1 {
		logsDir = os.Args[1]
	}

	fmt.Printf("🔍 Analyzing AI request/response logs in: %s\n\n", logsDir)

	// Find all request files
	reqFiles, err := filepath.Glob(filepath.Join(logsDir, "*.req.json"))
	if err != nil {
		fmt.Printf("❌ Error finding log files: %v\n", err)
		os.Exit(1)
	}

	if len(reqFiles) == 0 {
		fmt.Printf("⚠️  No log files found in %s\n", logsDir)
		fmt.Printf("💡 Logs will be created automatically when the coordinator makes AI requests.\n")
		os.Exit(0)
	}

	// Sort by number
	sort.Slice(reqFiles, func(i, j int) bool {
		numI := extractNumber(reqFiles[i])
		numJ := extractNumber(reqFiles[j])
		return numI < numJ
	})

	fmt.Printf("📊 Found %d request/response pairs\n\n", len(reqFiles))
	fmt.Println("=" + strings.Repeat("=", 79))

	// Analyze each pair
	totalMessages := 0
	totalToolCalls := 0
	totalErrors := 0

	for _, reqFile := range reqFiles {
		num := extractNumber(reqFile)
		resFile := filepath.Join(logsDir, fmt.Sprintf("%d.res.json", num))

		fmt.Printf("\n🔸 Request/Response Pair #%d\n", num)
		fmt.Println(strings.Repeat("-", 80))

		// Analyze request
		req, err := analyzeRequest(reqFile)
		if err != nil {
			fmt.Printf("❌ Error reading request: %v\n", err)
			continue
		}

		totalMessages += len(req.MsgContents)

		// Analyze response
		res, err := analyzeResponse(resFile)
		if err != nil {
			fmt.Printf("❌ Error reading response: %v\n", err)
			continue
		}

		if res.Error != "" {
			totalErrors++
		}

		// Extract tool calls from response
		toolCalls := extractToolCalls(res.Response)
		totalToolCalls += toolCalls

		// Calculate timing
		reqTime, _ := time.Parse(time.RFC3339Nano, req.Timestamp)
		resTime, _ := time.Parse(time.RFC3339Nano, res.Timestamp)
		duration := resTime.Sub(reqTime)

		// Print summary
		fmt.Printf("  Provider: %s\n", req.Provider)
		fmt.Printf("  Model: %v\n", req.Options["model"])
		fmt.Printf("  Messages in history: %d\n", len(req.MsgContents))
		fmt.Printf("  Tools available: %d\n", len(req.Tools))
		fmt.Printf("  Response time: %v\n", duration)
		fmt.Printf("  Tool calls made: %d\n", toolCalls)

		if res.Error != "" {
			fmt.Printf("  ❌ Error: %s\n", res.Error)
		}
	}

	// Print overall statistics
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("\n📈 Overall Statistics:")
	fmt.Printf("  Total requests: %d\n", len(reqFiles))
	fmt.Printf("  Total messages processed: %d\n", totalMessages)
	fmt.Printf("  Total tool calls made: %d\n", totalToolCalls)
	fmt.Printf("  Total errors: %d\n", totalErrors)
	fmt.Printf("  Success rate: %.1f%%\n", float64(len(reqFiles)-totalErrors)/float64(len(reqFiles))*100)
	fmt.Println()
}

func extractNumber(filename string) int {
	base := filepath.Base(filename)
	numStr := strings.TrimSuffix(base, ".req.json")
	num, _ := strconv.Atoi(numStr)
	return num
}

func analyzeRequest(filename string) (*RequestLog, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var req RequestLog
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}

	return &req, nil
}

func analyzeResponse(filename string) (*ResponseLog, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var res ResponseLog
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func extractToolCalls(response map[string]interface{}) int {
	if response == nil {
		return 0
	}

	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return 0
	}

	count := 0
	for _, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			continue
		}

		toolCalls, ok := choiceMap["tool_calls"].([]interface{})
		if ok {
			count += len(toolCalls)
		}
	}

	return count
}
