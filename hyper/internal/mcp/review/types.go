package review

import (
	"sync"
	"time"
)

// ReferenceType defines the type of reference found in knowledge entries
type ReferenceType string

const (
	ReferenceTypeFileLine    ReferenceType = "file_line"    // file.go:123 or file.go (lines 10-20)
	ReferenceTypeFunction    ReferenceType = "function"     // FunctionName(), Type.Method()
	ReferenceTypeCommit      ReferenceType = "commit"       // Git commit hash (7 or 40 chars)
	ReferenceTypeAPI         ReferenceType = "api"          // POST /api/v1/endpoint
	ReferenceTypeFile        ReferenceType = "file"         // file path without line number
)

// Reference represents a detected reference in knowledge entry text
type Reference struct {
	Type         ReferenceType
	Value        string // The actual reference text (e.g., "handlers/mcp.go:246")
	Context      string // Surrounding text for context (up to 50 chars before/after)
	Validated    bool
	ErrorMessage string
}

// VerificationResult contains the results of verifying all references in an entry
type VerificationResult struct {
	TotalReferences   int           `json:"totalReferences"`
	ValidReferences   int           `json:"validReferences"`
	InvalidReferences []Reference   `json:"invalidReferences"`
	BrokenReferences  []Reference   `json:"brokenReferences"`
	ValidationTime    time.Duration `json:"validationTime"`
}

// ValidationCacheEntry represents a cached validation result
type ValidationCacheEntry struct {
	Result    bool
	Timestamp time.Time
}

// ValidationCache provides a thread-safe cache for validation results
type ValidationCache struct {
	cache map[string]*ValidationCacheEntry
	ttl   time.Duration
	mu    sync.RWMutex
}
