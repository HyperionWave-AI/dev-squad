package mcp

// Removed orphaned tests for non-existent functions (2025-11-19):
// - TestParseArrayParameter and related tests: parseArrayParameter() function does not exist
// - TestEstimateTokens: estimateTokens() is a private function in handlers package, not accessible from mcp package
//
// These tests were preventing compilation with "undefined" errors.
// If these functions are needed in the future, they should be:
// 1. Implemented in this package (mcp), OR
// 2. Exported from their source package and properly imported
