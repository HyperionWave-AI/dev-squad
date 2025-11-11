package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Test data structures matching the API
type AgentCommunicationRequest struct {
	AgentType         string                 `json:"agentType"`
	CommunicationType string                 `json:"communicationType"`
	Message           string                 `json:"message,omitempty"`
	TaskID            string                 `json:"taskId,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
}

type AgentCommunicationResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	AgentType string                 `json:"agentType"`
	Timestamp string                 `json:"timestamp"`
}

func main() {
	baseURL := "http://localhost:8080/api/v1/agents/communicate"
	
	fmt.Println("🧪 Testing Agent Communication API Endpoints")
	fmt.Println(strings.Repeat("=", 50))
	
	// Test cases for different agent types and communication types
	testCases := []struct {
		name        string
		request     AgentCommunicationRequest
		expectError bool
		description string
	}{
		// Valid agent types with status requests
		{
			name: "UI-Dev Status Request",
			request: AgentCommunicationRequest{
				AgentType:         "ui-dev",
				CommunicationType: "status",
			},
			expectError: false,
			description: "Test status request for ui-dev agent",
		},
		{
			name: "Go-Dev Status Request",
			request: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "status",
			},
			expectError: false,
			description: "Test status request for go-dev agent",
		},
		{
			name: "SRE Status Request",
			request: AgentCommunicationRequest{
				AgentType:         "sre",
				CommunicationType: "status",
			},
			expectError: false,
			description: "Test status request for sre agent",
		},
		{
			name: "Coordinator Status Request",
			request: AgentCommunicationRequest{
				AgentType:         "coordinator",
				CommunicationType: "status",
			},
			expectError: false,
			description: "Test status request for coordinator agent",
		},
		{
			name: "Data-Analyst Status Request",
			request: AgentCommunicationRequest{
				AgentType:         "data-analyst",
				CommunicationType: "status",
			},
			expectError: false,
			description: "Test status request for data-analyst agent",
		},
		{
			name: "QA Status Request",
			request: AgentCommunicationRequest{
				AgentType:         "qa",
				CommunicationType: "status",
			},
			expectError: false,
			description: "Test status request for qa agent",
		},
		
		// Direct message tests
		{
			name: "Direct Message to Go-Dev",
			request: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "direct_message",
				Message:           "Hello, this is a test direct message",
			},
			expectError: false,
			description: "Test direct message communication",
		},
		{
			name: "Direct Message to UI-Dev",
			request: AgentCommunicationRequest{
				AgentType:         "ui-dev",
				CommunicationType: "direct_message",
				Message:           "Test message for UI development",
			},
			expectError: false,
			description: "Test direct message to UI developer",
		},
		
		// Execute command tests
		{
			name: "Execute List Subagents",
			request: AgentCommunicationRequest{
				AgentType:         "coordinator",
				CommunicationType: "execute",
				Parameters: map[string]interface{}{
					"command": "list_subagents",
				},
			},
			expectError: false,
			description: "Test execute command for listing subagents",
		},
		{
			name: "Execute Query Knowledge",
			request: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "execute",
				Parameters: map[string]interface{}{
					"command":    "query_knowledge",
					"collection": "test-collection",
					"query":      "test query",
					"limit":      5,
				},
			},
			expectError: false,
			description: "Test execute command for knowledge query",
		},
		
		// Error cases - Invalid agent types
		{
			name: "Invalid Agent Type",
			request: AgentCommunicationRequest{
				AgentType:         "invalid-agent",
				CommunicationType: "status",
			},
			expectError: true,
			description: "Test invalid agent type rejection",
		},
		
		// Error cases - Invalid communication types
		{
			name: "Invalid Communication Type",
			request: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "invalid-comm-type",
			},
			expectError: true,
			description: "Test invalid communication type rejection",
		},
		
		// Error cases - Missing required fields
		{
			name: "Missing Message for Direct Message",
			request: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "direct_message",
				// Missing message field
			},
			expectError: true,
			description: "Test validation for missing message in direct_message",
		},
		{
			name: "Missing Command for Execute",
			request: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "execute",
				// Missing command in parameters
			},
			expectError: true,
			description: "Test validation for missing command in execute",
		},
	}
	
	// Run all test cases
	passedTests := 0
	totalTests := len(testCases)
	
	for i, testCase := range testCases {
		fmt.Printf("\n🔍 Test %d/%d: %s\n", i+1, totalTests, testCase.name)
		fmt.Printf("   Description: %s\n", testCase.description)
		
		success := runTest(baseURL, testCase.request, testCase.expectError)
		if success {
			passedTests++
			fmt.Printf("   ✅ PASSED\n")
		} else {
			fmt.Printf("   ❌ FAILED\n")
		}
	}
	
	// Summary
	fmt.Printf("\n" + strings.Repeat("=", 50))
	fmt.Printf("\n📊 Test Results Summary:")
	fmt.Printf("\n   Total Tests: %d", totalTests)
	fmt.Printf("\n   Passed: %d", passedTests)
	fmt.Printf("\n   Failed: %d", totalTests-passedTests)
	fmt.Printf("\n   Success Rate: %.1f%%\n", float64(passedTests)/float64(totalTests)*100)
	
	if passedTests == totalTests {
		fmt.Println("\n🎉 All tests passed! Agent communication API is working correctly.")
	} else {
		fmt.Printf("\n⚠️  %d test(s) failed. Please check the implementation.\n", totalTests-passedTests)
	}
}

func runTest(baseURL string, request AgentCommunicationRequest, expectError bool) bool {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("   Error marshaling request: %v\n", err)
		return false
	}
	
	// Create HTTP request
	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("   Error creating request: %v\n", err)
		return false
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Agent-Communication-Test/1.0")
	
	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   Error sending request: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	
	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("   Error reading response: %v\n", err)
		return false
	}
	
	// Parse response
	var response AgentCommunicationResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		fmt.Printf("   Error parsing response: %v\n", err)
		fmt.Printf("   Raw response: %s\n", string(responseBody))
		return false
	}
	
	// Log response details
	fmt.Printf("   Status Code: %d\n", resp.StatusCode)
	fmt.Printf("   Response Success: %t\n", response.Success)
	fmt.Printf("   Response Message: %s\n", response.Message)
	
	// Validate response based on expectations
	if expectError {
		// We expect this to fail
		if resp.StatusCode >= 400 && !response.Success {
			return true // Test passed - error was expected
		} else {
			fmt.Printf("   Expected error but got success\n")
			return false
		}
	} else {
		// We expect this to succeed
		if resp.StatusCode == 200 && response.Success {
			// Additional validation for successful responses
			if response.AgentType != request.AgentType {
				fmt.Printf("   Agent type mismatch: expected %s, got %s\n", request.AgentType, response.AgentType)
				return false
			}
			if response.Timestamp == "" {
				fmt.Printf("   Missing timestamp in response\n")
				return false
			}
			return true // Test passed
		} else {
			fmt.Printf("   Expected success but got error\n")
			return false
		}
	}
}