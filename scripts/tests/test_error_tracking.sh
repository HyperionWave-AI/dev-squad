#!/bin/bash

# Test script for error tracking and lesson suggestion

echo "Testing error tracking..."
echo

# Simulate first error occurrence
echo "1. Recording first error occurrence..."
curl -s -X POST http://localhost:4097/api/v1/reflection/test-error \
  -H "Content-Type: application/json" \
  -d '{
    "errorType": "database-connection",
    "message": "Failed to connect to MongoDB: connection timeout after 10s",
    "stackTrace": "at main.go:123\nat connection.go:456",
    "context": {"host": "localhost", "port": 27017}
  }' | jq .

echo
echo "2. Recording second error occurrence (should trigger suggestion)..."
curl -s -X POST http://localhost:4097/api/v1/reflection/test-error \
  -H "Content-Type: application/json" \
  -d '{
    "errorType": "database-connection",
    "message": "Failed to connect to MongoDB: connection timeout after 10s",
    "stackTrace": "at main.go:123\nat connection.go:456",
    "context": {"host": "localhost", "port": 27017}
  }' | jq .

echo
echo "3. Recording third error occurrence..."
curl -s -X POST http://localhost:4097/api/v1/reflection/test-error \
  -H "Content-Type": "application/json" \
  -d '{
    "errorType": "database-connection",
    "message": "Failed to connect to MongoDB: connection timeout after 10s",
    "stackTrace": "at main.go:123\nat connection.go:456",
    "context": {"host": "localhost", "port": 27017}
  }' | jq .
