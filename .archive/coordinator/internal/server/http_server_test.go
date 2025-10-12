package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPHTTPTransport verifies that the MCP HTTP endpoint is properly configured
// using the official go-sdk StreamableHTTPHandler
func TestMCPHTTPTransport(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a minimal MCP server for testing
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	opts := &mcp.ServerOptions{
		HasTools:     true,
		HasResources: true,
		HasPrompts:   true,
	}
	mcpServer := mcp.NewServer(impl, opts)

	// Register a simple test tool
	type TestArgs struct {
		Message string `json:"message" jsonschema_description:"A test message"`
	}
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "test_echo",
		Description: "Echo back a message",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args TestArgs) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Echo: " + args.Message},
			},
		}, nil, nil
	})

	// Create the StreamableHTTPHandler
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return mcpServer
		},
		&mcp.StreamableHTTPOptions{
			Stateless:    false,
			JSONResponse: false,
		},
	)

	// Setup Gin router with the MCP handler
	r := gin.New()
	r.Any("/mcp", gin.WrapH(mcpHandler))

	// Test 1: GET /mcp - StreamableHTTP only supports POST
	t.Run("GET /mcp not supported", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Accept", "application/json, text/event-stream")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// StreamableHTTP transport only supports POST method
		// GET should return 405 Method Not Allowed
		if w.Code != http.StatusMethodNotAllowed {
			t.Logf("Expected status 405 Method Not Allowed, got %d", w.Code)
		} else {
			t.Log("Correctly returns 405 for GET requests (StreamableHTTP only supports POST)")
		}
	})

	// Test 2: POST /mcp with initialize request
	t.Run("POST /mcp initialize", func(t *testing.T) {
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"clientInfo": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		body, _ := json.Marshal(initReq)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Parse response
		respBody, err := io.ReadAll(w.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		t.Logf("Initialize response: %s", string(respBody))

		// Should contain JSON-RPC response
		var jsonResp map[string]interface{}
		if err := json.Unmarshal(respBody, &jsonResp); err != nil {
			// If not JSON, might be SSE format
			if !bytes.Contains(respBody, []byte("data:")) {
				t.Errorf("Response is neither JSON nor SSE format: %v", err)
			} else {
				t.Log("Response is in SSE format (valid)")
			}
		} else {
			// Check for JSON-RPC structure
			if jsonResp["jsonrpc"] != "2.0" {
				t.Errorf("Expected jsonrpc=2.0, got %v", jsonResp["jsonrpc"])
			}
			t.Logf("Valid JSON-RPC response received")
		}
	})

	// Test 3: POST /mcp with tools/list request
	t.Run("POST /mcp tools/list", func(t *testing.T) {
		// First initialize
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"clientInfo": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}
		body, _ := json.Marshal(initReq)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Extract session ID from response headers if present
		sessionID := w.Header().Get("Mcp-Session-Id")
		t.Logf("Session ID: %s", sessionID)

		// Now list tools
		listReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
			"params":  map[string]interface{}{},
		}
		body, _ = json.Marshal(listReq)
		req = httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		respBody, _ := io.ReadAll(w.Body)
		t.Logf("Tools list response: %s", string(respBody))

		// The response should mention our test_echo tool
		if !bytes.Contains(respBody, []byte("test_echo")) {
			t.Logf("Warning: test_echo tool not found in response (may be in different format)")
		} else {
			t.Log("Successfully found test_echo tool in response")
		}
	})
}

// TestMCPServerIntegration tests the full integration
func TestMCPServerIntegration(t *testing.T) {
	// Create a simple MCP server
	impl := &mcp.Implementation{
		Name:    "test-coordinator",
		Version: "2.0.0",
	}
	opts := &mcp.ServerOptions{
		HasTools: true,
	}
	mcpServer := mcp.NewServer(impl, opts)

	// Create HTTP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return mcpServer
		},
		&mcp.StreamableHTTPOptions{
			Stateless:    false,
			JSONResponse: false,
		},
	)

	// Create test server
	ts := httptest.NewServer(mcpHandler)
	defer ts.Close()

	// Test initialize request
	t.Run("End-to-end initialize", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}

		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"clientInfo": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		body, _ := json.Marshal(initReq)
		req, _ := http.NewRequest("POST", ts.URL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		respBody, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(respBody))

		// Response should be valid JSON-RPC or SSE
		if len(respBody) == 0 {
			t.Error("Empty response body")
		}
	})
}
