package parser

import (
	"fmt"
	"path/filepath"
	"sync"
)

// NodeType represents the type of AST node (function, class, method, etc.)
type NodeType string

const (
	NodeTypeFunction      NodeType = "function"
	NodeTypeMethod        NodeType = "method"
	NodeTypeClass         NodeType = "class"
	NodeTypeInterface     NodeType = "interface"
	NodeTypeStruct        NodeType = "struct"
	NodeTypeType          NodeType = "type"
	NodeTypeVariable      NodeType = "variable"
	NodeTypeConstant      NodeType = "constant"
	NodeTypeImport        NodeType = "import"
	NodeTypeComment       NodeType = "comment"
	NodeTypeArrowFunction NodeType = "arrow_function"
)

// CodeNode represents a parsed AST node with semantic information
type CodeNode struct {
	Type      NodeType               // Type of node (function, class, method, etc.)
	Name      string                 // Name of the node (function name, class name, etc.)
	StartLine int                    // Starting line number in source file
	EndLine   int                    // Ending line number in source file
	Content   string                 // Full source code content of the node
	Signature string                 // Function/method signature or type declaration
	Metadata  map[string]interface{} // Additional metadata (decorators, access modifiers, etc.)
}

// ASTParser is the interface that all language-specific parsers must implement
type ASTParser interface {
	// Parse parses a file and returns a list of code nodes
	// Returns error if parsing fails completely
	Parse(filePath string, content []byte) ([]CodeNode, error)

	// SupportsLanguage returns true if this parser supports the given language
	SupportsLanguage(language string) bool
}

// ParserRegistry manages the mapping between file extensions/languages and parsers
type ParserRegistry struct {
	mu                sync.RWMutex
	extensionParsers  map[string]ASTParser // .go -> GoParser
	languageParsers   map[string]ASTParser // go -> GoParser
}

var (
	// globalRegistry is the singleton parser registry
	globalRegistry *ParserRegistry
	registryOnce   sync.Once
)

// GetRegistry returns the global parser registry (singleton)
func GetRegistry() *ParserRegistry {
	registryOnce.Do(func() {
		globalRegistry = &ParserRegistry{
			extensionParsers: make(map[string]ASTParser),
			languageParsers:  make(map[string]ASTParser),
		}
	})
	return globalRegistry
}

// RegisterParser registers a parser for specific extensions and languages
func (r *ParserRegistry) RegisterParser(parser ASTParser, extensions []string, languages []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, ext := range extensions {
		// Ensure extension starts with a dot
		if ext != "" && ext[0] != '.' {
			ext = "." + ext
		}
		r.extensionParsers[ext] = parser
	}

	for _, lang := range languages {
		r.languageParsers[lang] = parser
	}
}

// GetParserForFile returns the appropriate parser for a given file path
func (r *ParserRegistry) GetParserForFile(filePath string) (ASTParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext := filepath.Ext(filePath)
	if parser, ok := r.extensionParsers[ext]; ok {
		return parser, nil
	}

	return nil, fmt.Errorf("no parser registered for extension: %s", ext)
}

// GetParserForLanguage returns the appropriate parser for a given language
func (r *ParserRegistry) GetParserForLanguage(language string) (ASTParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if parser, ok := r.languageParsers[language]; ok {
		return parser, nil
	}

	return nil, fmt.Errorf("no parser registered for language: %s", language)
}

// HasParserForFile checks if a parser is registered for the given file
func (r *ParserRegistry) HasParserForFile(filePath string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext := filepath.Ext(filePath)
	_, ok := r.extensionParsers[ext]
	return ok
}

// HasParserForLanguage checks if a parser is registered for the given language
func (r *ParserRegistry) HasParserForLanguage(language string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.languageParsers[language]
	return ok
}
