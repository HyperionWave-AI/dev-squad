package parser

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSParser_HelperBranches(t *testing.T) {
	p := NewJSParser()

	t.Run("findArrowFunctionEndWithBlockBody", func(t *testing.T) {
		lines := []string{
			"const blockFn = () => {",
			"  return value",
			"}",
		}

		assert.Equal(t, 2, p.findArrowFunctionEnd(lines, 0))
	})

	t.Run("findArrowFunctionEndWithExpressionContinuation", func(t *testing.T) {
		lines := []string{
			"const exprFn = () => sum(",
			"  left + right",
			"nextLine",
		}

		assert.Equal(t, 1, p.findArrowFunctionEnd(lines, 0))
	})

	t.Run("findArrowFunctionEndFallbackToStart", func(t *testing.T) {
		lines := []string{
			"const exprFn = () => first,",
			"  second,",
			"  third,",
			"  fourth,",
			"  fifth,",
			"  sixth,",
		}

		assert.Equal(t, 0, p.findArrowFunctionEnd(lines, 0))
	})

	t.Run("extractContentClampsBounds", func(t *testing.T) {
		lines := []string{"one", "two", "three"}
		content := p.extractContent(lines, -5, 99)
		assert.Equal(t, "one\ntwo\nthree", content)
	})

	t.Run("extractDocstringSingleLineComments", func(t *testing.T) {
		lines := []string{
			"// one",
			"// two",
			"function add() {}",
		}

		assert.Equal(t, "// one\n// two", p.extractDocstring(lines, 2))
	})

	t.Run("extractDocstringJsdocBlock", func(t *testing.T) {
		lines := []string{
			"/**",
			" * docs",
			" */",
			"function add() {}",
		}

		doc := p.extractDocstring(lines, 3)
		assert.Contains(t, doc, "docs")
		assert.Contains(t, doc, "/**")
	})

	t.Run("extractDocstringReturnsEmptyWithoutComment", func(t *testing.T) {
		lines := []string{
			"const x = 1",
			"function add() {}",
		}

		assert.Empty(t, p.extractDocstring(lines, 1))
	})

	t.Run("extractMethodsInRangeSkipsControlFlowKeywords", func(t *testing.T) {
		lines := []string{
			"class Sample {",
			"  if(condition) {",
			"    return 1;",
			"  }",
			"  async runTask() {",
			"    return work();",
			"  }",
			"}",
		}

		nodes := p.extractMethodsInRange(lines, "Sample", 0, len(lines)-1)
		if assert.Len(t, nodes, 1) {
			assert.Equal(t, "Sample.runTask", nodes[0].Name)
			assert.Equal(t, true, nodes[0].Metadata["async"])
		}
	})
}

func TestJSParser_Parse_UnclosedBlocksFallback(t *testing.T) {
	p := NewJSParser()

	code := "function open() {\nexport class Broken {\n"
	nodes, err := p.Parse("broken.js", []byte(code))
	assert.NoError(t, err)

	var functionNode *CodeNode
	var classNode *CodeNode
	for i := range nodes {
		switch nodes[i].Name {
		case "open":
			functionNode = &nodes[i]
		case "Broken":
			classNode = &nodes[i]
		}
	}

	if assert.NotNil(t, functionNode) {
		assert.Equal(t, functionNode.StartLine, functionNode.EndLine)
	}
	if assert.NotNil(t, classNode) {
		assert.Equal(t, classNode.StartLine, classNode.EndLine)
	}
}

func TestJSParser_Parse_UnclosedArrowFunctionFallback(t *testing.T) {
	p := NewJSParser()

	code := "// docs\nconst openArrow = (x) => {\n"
	nodes, err := p.Parse("arrow.js", []byte(code))
	assert.NoError(t, err)

	var arrowNode *CodeNode
	for i := range nodes {
		if nodes[i].Type == NodeTypeArrowFunction {
			arrowNode = &nodes[i]
			break
		}
	}

	if assert.NotNil(t, arrowNode) {
		assert.Equal(t, arrowNode.StartLine, arrowNode.EndLine)
		assert.Equal(t, true, arrowNode.Metadata["arrow_function"])
	}
}

func TestGoParser_HelperBranches(t *testing.T) {
	p := NewGoParser()

	t.Run("extractReceiverTypeVariants", func(t *testing.T) {
		ident := &ast.Field{Type: ast.NewIdent("Worker")}
		ptr := &ast.Field{Type: &ast.StarExpr{X: ast.NewIdent("Worker")}}
		unknown := &ast.Field{
			Type: &ast.StarExpr{
				X: &ast.SelectorExpr{
					X:   ast.NewIdent("pkg"),
					Sel: ast.NewIdent("Worker"),
				},
			},
		}

		assert.Equal(t, "Worker", p.extractReceiverType(ident))
		assert.Equal(t, "*Worker", p.extractReceiverType(ptr))
		assert.Equal(t, "unknown", p.extractReceiverType(unknown))
	})

	t.Run("buildTypeSignatureDefaultBranch", func(t *testing.T) {
		spec := &ast.TypeSpec{Name: ast.NewIdent("Alias"), Type: ast.NewIdent("int")}
		sig := p.buildTypeSignature(spec, &ast.GenDecl{})
		assert.Equal(t, "type Alias ...", sig)
	})

	t.Run("buildValueSignatureConstAndVarMultipleNames", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("A"), ast.NewIdent("B")},
		}

		constSig := p.buildValueSignature(spec, &ast.GenDecl{Tok: token.CONST})
		varSig := p.buildValueSignature(spec, &ast.GenDecl{Tok: token.VAR})

		assert.Equal(t, "const A, B", constSig)
		assert.Equal(t, "var A, B", varSig)
	})
}

func TestGoParser_Parse_IgnoresImportsAndUnexportedValues(t *testing.T) {
	p := NewGoParser()

	code := `package main

import "fmt"

var hidden = 1
var Visible = 2
const hiddenConst = 3
const VisibleConst = 4

func use() string {
	return fmt.Sprintf("%d", Visible)
}`

	nodes, err := p.Parse("sample.go", []byte(code))
	assert.NoError(t, err)

	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		assert.NotEqual(t, NodeTypeImport, node.Type)
		names = append(names, node.Name)
	}

	assert.Contains(t, names, "Visible")
	assert.Contains(t, names, "VisibleConst")
	assert.NotContains(t, names, "hidden")
	assert.NotContains(t, names, "hiddenConst")
}

func TestPythonParser_HelperBranches(t *testing.T) {
	p := NewPythonParser()

	lines := []string{
		"class Top:",
		"    def first(self):",
		"        return 1",
		"",
		"    # comment",
		"    def second(self):",
		"        return 2",
		"def outside():",
		"    return 3",
		"    class Nested:",
		"        pass",
		"class Final:",
		"    pass",
	}

	t.Run("findClassRangesOnlyTopLevel", func(t *testing.T) {
		ranges := p.findClassRanges(lines)
		if assert.Len(t, ranges, 2) {
			assert.Equal(t, "Top", ranges[0].name)
			assert.Equal(t, "Final", ranges[1].name)
		}
	})

	t.Run("isWithinClassRange", func(t *testing.T) {
		ranges := []pythonClassRange{
			{name: "Top", startLine: 1, endLine: 7},
		}
		assert.True(t, p.isWithinClassRange(4, ranges))
		assert.False(t, p.isWithinClassRange(8, ranges))
	})

	t.Run("findPythonBlockEndOutOfRange", func(t *testing.T) {
		assert.Equal(t, -1, p.findPythonBlockEnd(lines, len(lines)))
	})

	t.Run("getIndentLevelHandlesTabs", func(t *testing.T) {
		assert.Equal(t, 4, p.getIndentLevel("\tdef run(self):"))
	})

	t.Run("extractContentClampsBounds", func(t *testing.T) {
		content := p.extractContent(lines, -3, 99)
		assert.Contains(t, content, "class Top:")
		assert.Contains(t, content, "class Final:")
	})

	t.Run("extractMethodsInRangeSkipsTopLevelAndTruncatesAtEndIdx", func(t *testing.T) {
		methodLines := []string{
			"def top_level(self):",
			"    return 0",
			"    def nested(self):",
			"        return 1",
		}

		skipped := p.extractMethodsInRange(methodLines, "Top", 0, 1)
		assert.Empty(t, skipped)

		truncatedLines := []string{
			"    def run(self):",
			"        value = 1",
			"        return value",
		}
		truncated := p.extractMethodsInRange(truncatedLines, "Top", 0, 0)
		if assert.Len(t, truncated, 1) {
			assert.Equal(t, 1, truncated[0].EndLine)
		}
	})

	t.Run("buildMethodSignatureAsyncAndSync", func(t *testing.T) {
		asyncSig := p.buildMethodSignature("Top", "run", true, []string{"classmethod"})
		syncSig := p.buildMethodSignature("Top", "run", false, nil)

		assert.Equal(t, "@classmethod async def Top.run(...)", asyncSig)
		assert.Equal(t, "def Top.run(...)", syncSig)
	})
}
