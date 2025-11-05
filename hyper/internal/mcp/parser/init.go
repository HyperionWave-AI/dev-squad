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

		// Register Tree-sitter parsers for additional languages
		// Java
		javaParser, _ := NewTreeSitterParser("java")
		if javaParser != nil {
			registry.RegisterParser(
				javaParser,
				[]string{".java"},
				[]string{"java"},
			)
		}

		// C++
		cppParser, _ := NewTreeSitterParser("cpp")
		if cppParser != nil {
			registry.RegisterParser(
				cppParser,
				[]string{".cpp", ".cc", ".cxx", ".c++", ".hpp", ".h"},
				[]string{"cpp", "c++"},
			)
		}

		// Rust
		rustParser, _ := NewTreeSitterParser("rust")
		if rustParser != nil {
			registry.RegisterParser(
				rustParser,
				[]string{".rs"},
				[]string{"rust", "rs"},
			)
		}
	})
}
