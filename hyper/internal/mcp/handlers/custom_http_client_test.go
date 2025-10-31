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
