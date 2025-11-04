package parser

import "sync"

var (
	initOnce sync.Once
)

// InitializeParsers registers all available parsers
// This should be called at application startup
func InitializeParsers() {
	initOnce.Do(func() {
		registry := GetRegistry()

		// Register Go parser
		goParser := NewGoParser()
		registry.RegisterParser(
			goParser,
			[]string{".go"},
			[]string{"go", "golang"},
		)

		// Register JavaScript/TypeScript parser
		jsParser := NewJSParser()
		registry.RegisterParser(
			jsParser,
			[]string{".js", ".ts", ".jsx", ".tsx"},
			[]string{"javascript", "typescript", "js", "ts", "jsx", "tsx"},
		)

		// Register Python parser
		pyParser := NewPythonParser()
		registry.RegisterParser(
			pyParser,
			[]string{".py"},
			[]string{"python", "py"},
		)
	})
}
