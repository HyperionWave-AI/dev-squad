package aiservice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// HTTPLogger logs raw HTTP requests and responses to AI providers
type HTTPLogger struct {
	logDir  string
	counter atomic.Int64
	enabled bool
}

// NewHTTPLogger creates a new HTTP logger that writes to the specified directory
func NewHTTPLogger(logDir string) *HTTPLogger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to create log directory %s: %v\n", logDir, err)
		return &HTTPLogger{enabled: false}
	}

	return &HTTPLogger{
		logDir:  logDir,
		enabled: true,
	}
}

// LogRequest logs an HTTP request to a numbered JSON file
func (l *HTTPLogger) LogRequest(provider string, messages interface{}, tools interface{}, options map[string]interface{}) {
	if !l.enabled {
		return
	}

	num := l.counter.Add(1)
	filename := filepath.Join(l.logDir, fmt.Sprintf("%d.req.json", num))

	request := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"provider":  provider,
		"messages":  messages,
		"tools":     tools,
		"options":   options,
	}

	data, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to marshal request: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to write request file %s: %v\n", filename, err)
	} else {
		fmt.Printf("[HTTP Logger] ✓ Logged request to %s\n", filename)
	}
}

// LogResponse logs an HTTP response to a numbered JSON file
func (l *HTTPLogger) LogResponse(provider string, response interface{}, err error) {
	if !l.enabled {
		return
	}

	num := l.counter.Load()
	filename := filepath.Join(l.logDir, fmt.Sprintf("%d.res.json", num))

	result := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"provider":  provider,
		"response":  response,
	}

	if err != nil {
		result["error"] = err.Error()
	}

	data, jsonErr := json.MarshalIndent(result, "", "  ")
	if jsonErr != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to marshal response: %v\n", jsonErr)
		return
	}

	if writeErr := os.WriteFile(filename, data, 0644); writeErr != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to write response file %s: %v\n", filename, writeErr)
	} else {
		fmt.Printf("[HTTP Logger] ✓ Logged response to %s\n", filename)
	}
}

// LogLangChainRequest logs a LangChain request with msgContents and tools
func (l *HTTPLogger) LogLangChainRequest(provider string, msgContents []interface{}, tools []interface{}, options map[string]interface{}) {
	if !l.enabled {
		return
	}

	num := l.counter.Add(1)
	filename := filepath.Join(l.logDir, fmt.Sprintf("%d.req.json", num))

	request := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"provider":    provider,
		"msgContents": msgContents,
		"tools":       tools,
		"options":     options,
	}

	data, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to marshal LangChain request: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to write request file %s: %v\n", filename, err)
	} else {
		fmt.Printf("[HTTP Logger] ✓ Logged LangChain request to %s\n", filename)
	}
}

// LogLangChainResponse logs a LangChain response
func (l *HTTPLogger) LogLangChainResponse(provider string, response interface{}, err error) {
	if !l.enabled {
		return
	}

	num := l.counter.Load()
	filename := filepath.Join(l.logDir, fmt.Sprintf("%d.res.json", num))

	result := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"provider":  provider,
		"response":  response,
	}

	if err != nil {
		result["error"] = err.Error()
	}

	data, jsonErr := json.MarshalIndent(result, "", "  ")
	if jsonErr != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to marshal LangChain response: %v\n", jsonErr)
		return
	}

	if writeErr := os.WriteFile(filename, data, 0644); writeErr != nil {
		fmt.Printf("[HTTP Logger] Warning: Failed to write response file %s: %v\n", filename, writeErr)
	} else {
		fmt.Printf("[HTTP Logger] ✓ Logged LangChain response to %s\n", filename)
	}
}
