package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHeaderRoundTripper verifies that custom headers are properly injected into HTTP requests
func TestHeaderRoundTripper(t *testing.T) {
	// Create a test server that echoes back the Authorization header
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(authHeader))
	}))
	defer testServer.Close()

	// Test with headers
	headers := map[string]interface{}{
		"Authorization": "Bearer test-token-12345",
	}

	client := createCustomHTTPClient(headers)
	resp, err := client.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	expected := "Bearer test-token-12345"
	if string(body) != expected {
		t.Errorf("Expected Authorization header '%s', got '%s'", expected, string(body))
	}
}

// TestHeaderRoundTripperMultipleHeaders verifies multiple headers are injected
func TestHeaderRoundTripperMultipleHeaders(t *testing.T) {
	// Create a test server that echoes back multiple headers
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		apiKey := r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(auth + "|" + apiKey))
	}))
	defer testServer.Close()

	// Test with multiple headers
	headers := map[string]interface{}{
		"Authorization": "Bearer token123",
		"X-API-Key":     "api-key-456",
	}

	client := createCustomHTTPClient(headers)
	resp, err := client.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	expected := "Bearer token123|api-key-456"
	if string(body) != expected {
		t.Errorf("Expected headers '%s', got '%s'", expected, string(body))
	}
}

// TestCreateCustomHTTPClientNilHeaders verifies that nil headers returns default client
func TestCreateCustomHTTPClientNilHeaders(t *testing.T) {
	client := createCustomHTTPClient(nil)
	if client != http.DefaultClient {
		t.Error("Expected http.DefaultClient when headers are nil")
	}
}

// TestCreateCustomHTTPClientEmptyHeaders verifies that empty headers returns default client
func TestCreateCustomHTTPClientEmptyHeaders(t *testing.T) {
	client := createCustomHTTPClient(map[string]interface{}{})
	if client != http.DefaultClient {
		t.Error("Expected http.DefaultClient when headers are empty")
	}
}

// TestHeaderRoundTripperNonStringValue verifies non-string values are skipped
func TestHeaderRoundTripperNonStringValue(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		numHeader := r.Header.Get("X-Number")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(auth + "|" + numHeader))
	}))
	defer testServer.Close()

	// Test with mixed value types
	headers := map[string]interface{}{
		"Authorization": "Bearer token123",
		"X-Number":      12345, // Non-string value should be skipped
	}

	client := createCustomHTTPClient(headers)
	resp, err := client.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Only string header should be present, non-string should be empty
	expected := "Bearer token123|"
	if string(body) != expected {
		t.Errorf("Expected headers '%s', got '%s'", expected, string(body))
	}
}

// TestHeaderRoundTripperAcceptMerge verifies that Accept headers are merged, not replaced
// This is critical for MCP SDK compatibility which requires "application/json, text/event-stream"
func TestHeaderRoundTripperAcceptMerge(t *testing.T) {
	tests := []struct {
		name           string
		existingAccept string
		customAccept   string
		expectedAccept string
	}{
		{
			name:           "merge custom accept with SDK accept",
			existingAccept: "application/json, text/event-stream",
			customAccept:   "application/json",
			expectedAccept: "application/json, text/event-stream", // No duplicate added
		},
		{
			name:           "add new accept type",
			existingAccept: "application/json, text/event-stream",
			customAccept:   "application/xml",
			expectedAccept: "application/json, text/event-stream, application/xml",
		},
		{
			name:           "case insensitive duplicate detection",
			existingAccept: "application/json, text/event-stream",
			customAccept:   "APPLICATION/JSON",
			expectedAccept: "application/json, text/event-stream",
		},
		{
			name:           "preserve SDK required types when custom only has json",
			existingAccept: "application/json, text/event-stream",
			customAccept:   "application/json",
			expectedAccept: "application/json, text/event-stream",
		},
		{
			name:           "empty existing accept uses custom",
			existingAccept: "",
			customAccept:   "application/json",
			expectedAccept: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedAccept string

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedAccept = r.Header.Get("Accept")
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			// Create custom headers with Accept
			headers := map[string]interface{}{
				"Accept": tt.customAccept,
			}

			client := createCustomHTTPClient(headers)

			// Create request with existing Accept header (simulating MCP SDK)
			req, err := http.NewRequest("GET", testServer.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if tt.existingAccept != "" {
				req.Header.Set("Accept", tt.existingAccept)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()

			if receivedAccept != tt.expectedAccept {
				t.Errorf("Expected Accept header '%s', got '%s'", tt.expectedAccept, receivedAccept)
			}
		})
	}
}

// TestParseAcceptValues verifies the Accept header parsing helper
func TestParseAcceptValues(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "application/json, text/event-stream",
			expected: []string{"application/json", "text/event-stream"},
		},
		{
			input:    "application/json,text/event-stream",
			expected: []string{"application/json", "text/event-stream"},
		},
		{
			input:    "  application/json  ,  text/event-stream  ",
			expected: []string{"application/json", "text/event-stream"},
		},
		{
			input:    "application/json",
			expected: []string{"application/json"},
		},
		{
			input:    "",
			expected: nil,
		},
		{
			input:    "  ,  ,  ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAcceptValues(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d values, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("At index %d: expected '%s', got '%s'", i, tt.expected[i], v)
				}
			}
		})
	}
}
