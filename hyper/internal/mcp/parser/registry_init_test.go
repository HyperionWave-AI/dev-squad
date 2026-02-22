package parser

import (
	"sync"
	"testing"
)

type mockParser struct {
	language string
}

func (m *mockParser) Parse(filePath string, content []byte) ([]CodeNode, error) {
	return []CodeNode{{Type: NodeTypeFunction, Name: "mock"}}, nil
}

func (m *mockParser) SupportsLanguage(language string) bool {
	return language == m.language
}

func resetParserSingletonsForTest() {
	globalRegistry = nil
	registryOnce = sync.Once{}
	initOnce = sync.Once{}
}

func TestParserRegistry_BasicOperations(t *testing.T) {
	resetParserSingletonsForTest()
	reg := GetRegistry()
	if reg == nil {
		t.Fatal("expected registry")
	}

	p := &mockParser{language: "mock"}
	reg.RegisterParser(p, []string{"mock", ".mk"}, []string{"mock"})

	if !reg.HasParserForFile("file.mock") {
		t.Fatal("expected parser for .mock extension")
	}
	if !reg.HasParserForFile("file.mk") {
		t.Fatal("expected parser for .mk extension")
	}
	if reg.HasParserForFile("file.unknown") {
		t.Fatal("expected no parser for unknown extension")
	}

	fileParser, err := reg.GetParserForFile("file.mock")
	if err != nil {
		t.Fatalf("GetParserForFile returned error: %v", err)
	}
	if fileParser != p {
		t.Fatal("expected registered parser for file extension")
	}

	langParser, err := reg.GetParserForLanguage("mock")
	if err != nil {
		t.Fatalf("GetParserForLanguage returned error: %v", err)
	}
	if langParser != p {
		t.Fatal("expected registered parser for language")
	}
	if !reg.HasParserForLanguage("mock") {
		t.Fatal("expected parser for language")
	}
	if reg.HasParserForLanguage("unknown") {
		t.Fatal("expected no parser for unknown language")
	}

	if _, err := reg.GetParserForFile("file.none"); err == nil {
		t.Fatal("expected error for unknown extension")
	}
	if _, err := reg.GetParserForLanguage("none"); err == nil {
		t.Fatal("expected error for unknown language")
	}
}

func TestGetRegistry_Singleton(t *testing.T) {
	resetParserSingletonsForTest()
	r1 := GetRegistry()
	r2 := GetRegistry()
	if r1 != r2 {
		t.Fatal("expected GetRegistry to return singleton instance")
	}
}

func TestInitializeParsers_IdempotentAndRegistersDefaults(t *testing.T) {
	resetParserSingletonsForTest()

	InitializeParsers()
	InitializeParsers() // initOnce should prevent duplicate init panic/side effects

	reg := GetRegistry()
	for _, file := range []string{"a.go", "b.py", "c.ts", "d.jsx"} {
		if !reg.HasParserForFile(file) {
			t.Fatalf("expected default parser for %s after InitializeParsers", file)
		}
	}
}

func TestTreeSitterHelpers(t *testing.T) {
	p := &TreeSitterParser{languageName: "java"}
	if !p.SupportsLanguage("java") {
		t.Fatal("expected SupportsLanguage to match parser language")
	}
	if p.SupportsLanguage("definitely-unsupported") {
		t.Fatal("expected SupportsLanguage to reject unsupported languages")
	}

	if _, err := NewTreeSitterParser("definitely-unsupported"); err == nil {
		t.Fatal("expected unsupported language error from NewTreeSitterParser")
	}

	if got := GetLanguageFromExtension("src/main.java"); got != "java" {
		t.Fatalf("expected java for .java extension, got %q", got)
	}
	if got := GetLanguageFromExtension("src/main.unknownext"); got != "" {
		t.Fatalf("expected empty language for unknown extension, got %q", got)
	}
}
