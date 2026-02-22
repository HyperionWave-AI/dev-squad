package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func writeExecutable(t *testing.T, dir, name, script string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", name, err)
	}
	return path
}

func newValidator(t *testing.T, projectRoot string) *CodeValidator {
	t.Helper()
	return NewCodeValidator(zap.NewNop(), projectRoot)
}

func TestNewCodeValidator(t *testing.T) {
	projectRoot := t.TempDir()
	v := newValidator(t, projectRoot)

	if v == nil {
		t.Fatal("expected validator")
	}
	if v.projectRoot != projectRoot {
		t.Fatalf("expected projectRoot %q, got %q", projectRoot, v.projectRoot)
	}
}

func TestValidateFiles_Empty(t *testing.T) {
	v := newValidator(t, t.TempDir())

	result, err := v.ValidateFiles(context.Background(), nil)
	if err != nil {
		t.Fatalf("ValidateFiles returned error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected Passed=true for empty input")
	}
}

func TestValidateTypeScript_FiltersToModifiedFiles(t *testing.T) {
	projectRoot := t.TempDir()
	uiSrc := filepath.Join(projectRoot, "ui", "src")
	if err := os.MkdirAll(uiSrc, 0o755); err != nil {
		t.Fatalf("mkdir ui src: %v", err)
	}

	changedFile := filepath.Join(uiSrc, "changed.ts")
	otherFile := filepath.Join(uiSrc, "other.ts")
	if err := os.WriteFile(changedFile, []byte("const a = 1;\n"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
	if err := os.WriteFile(otherFile, []byte("const b = 2;\n"), 0o644); err != nil {
		t.Fatalf("write other file: %v", err)
	}

	binDir := t.TempDir()
	writeExecutable(t, binDir, "npx", `#!/bin/sh
echo "src/changed.ts:3:1 - error TS1005: ';' expected." 1>&2
echo "src/other.ts:9:2 - error TS2304: Cannot find name 'x'." 1>&2
exit 1
`)
	t.Setenv("PATH", binDir)

	v := newValidator(t, projectRoot)
	result, err := v.validateTypeScript(context.Background(), []string{changedFile})
	if err != nil {
		t.Fatalf("validateTypeScript returned error: %v", err)
	}

	if result.Passed {
		t.Fatal("expected Passed=false when modified file has error")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected exactly 1 relevant error, got %d", len(result.Errors))
	}
	if result.Errors[0].File != "src/changed.ts" {
		t.Fatalf("expected filtered file src/changed.ts, got %q", result.Errors[0].File)
	}
}

func TestValidateGo_ParsesVetOutput(t *testing.T) {
	projectRoot := t.TempDir()
	goDir := filepath.Join(projectRoot, "internal", "pkg")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatalf("mkdir go dir: %v", err)
	}
	goFile := filepath.Join(goDir, "file.go")
	if err := os.WriteFile(goFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	binDir := t.TempDir()
	writeExecutable(t, binDir, "go", `#!/bin/sh
if [ "$1" = "vet" ]; then
  echo "./internal/pkg/file.go:12: undefined: nope" 1>&2
  exit 1
fi
exit 0
`)
	t.Setenv("PATH", binDir)

	v := newValidator(t, projectRoot)
	result, err := v.validateGo(context.Background(), []string{goFile})
	if err != nil {
		t.Fatalf("validateGo returned error: %v", err)
	}

	if result.Passed {
		t.Fatal("expected Passed=false when go vet emits errors")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 parsed go error, got %d", len(result.Errors))
	}
	if result.Errors[0].Line != 12 {
		t.Fatalf("expected parsed line 12, got %d", result.Errors[0].Line)
	}
}

func TestValidatePython_SkipsWhenInterpreterMissing(t *testing.T) {
	projectRoot := t.TempDir()
	emptyBin := t.TempDir()
	t.Setenv("PATH", emptyBin)

	v := newValidator(t, projectRoot)
	result, err := v.validatePython(context.Background(), []string{"script.py"})
	if err != nil {
		t.Fatalf("validatePython returned error: %v", err)
	}

	if !result.Skipped {
		t.Fatal("expected python validation to be skipped")
	}
	if !result.Passed {
		t.Fatal("expected skipped validation to be treated as passed")
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestValidatePython_ParsesCompileError(t *testing.T) {
	projectRoot := t.TempDir()
	pyFile := filepath.Join(projectRoot, "broken.py")
	if err := os.WriteFile(pyFile, []byte("print(\n"), 0o644); err != nil {
		t.Fatalf("write python file: %v", err)
	}

	binDir := t.TempDir()
	writeExecutable(t, binDir, "python3", `#!/bin/sh
echo "  File \"$3\", line 7" 1>&2
echo "SyntaxError: invalid syntax" 1>&2
exit 1
`)
	t.Setenv("PATH", binDir)

	v := newValidator(t, projectRoot)
	result, err := v.validatePython(context.Background(), []string{pyFile})
	if err != nil {
		t.Fatalf("validatePython returned error: %v", err)
	}

	if result.Passed {
		t.Fatal("expected Passed=false for syntax errors")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 python error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != "PY_COMPILE" {
		t.Fatalf("expected code PY_COMPILE, got %q", result.Errors[0].Code)
	}
	if !strings.Contains(result.Errors[0].Message, "line 7") {
		t.Fatalf("expected line marker in python error message, got %q", result.Errors[0].Message)
	}
}

func TestValidateFiles_AggregatesByLanguage(t *testing.T) {
	projectRoot := t.TempDir()

	uiSrc := filepath.Join(projectRoot, "ui", "src")
	goDir := filepath.Join(projectRoot, "internal", "pkg")
	if err := os.MkdirAll(uiSrc, 0o755); err != nil {
		t.Fatalf("mkdir ui src: %v", err)
	}
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatalf("mkdir go dir: %v", err)
	}

	tsFile := filepath.Join(uiSrc, "target.ts")
	goFile := filepath.Join(goDir, "target.go")
	pyFile := filepath.Join(projectRoot, "ok.py")
	_ = os.WriteFile(tsFile, []byte("const x = 1;\n"), 0o644)
	_ = os.WriteFile(goFile, []byte("package pkg\n"), 0o644)
	_ = os.WriteFile(pyFile, []byte("print('ok')\n"), 0o644)

	binDir := t.TempDir()
	writeExecutable(t, binDir, "npx", `#!/bin/sh
echo "src/target.ts:1:1 - error TS2304: Cannot find name 'missing'." 1>&2
exit 1
`)
	writeExecutable(t, binDir, "go", `#!/bin/sh
echo "./internal/pkg/target.go:9: undefined: nope" 1>&2
exit 1
`)
	writeExecutable(t, binDir, "python3", `#!/bin/sh
exit 0
`)
	t.Setenv("PATH", binDir)

	v := newValidator(t, projectRoot)
	result, err := v.ValidateFiles(context.Background(), []string{tsFile, goFile, pyFile})
	if err != nil {
		t.Fatalf("ValidateFiles returned error: %v", err)
	}

	if result.Passed {
		t.Fatal("expected Passed=false when any validator reports errors")
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 combined errors, got %d", len(result.Errors))
	}
	if result.Command != "validated 3 files" {
		t.Fatalf("unexpected aggregate command: %q", result.Command)
	}
}

func TestParseTypeScriptOutputAndLineFormats(t *testing.T) {
	v := newValidator(t, t.TempDir())

	pretty := "src/a.ts:10:5 - error TS2304: Cannot find name 'foo'."
	nonPretty := "src/b.ts(8,2): warning TS6133: 'x' is declared but never read."
	junk := "some unrelated log line"
	output := strings.Join([]string{pretty, nonPretty, junk}, "\n")

	parsed := v.parseTypeScriptOutput(output)
	if len(parsed) != 2 {
		t.Fatalf("expected 2 parsed TypeScript entries, got %d", len(parsed))
	}

	if parsed[0].File != "src/a.ts" || parsed[0].Code != "TS2304" {
		t.Fatalf("unexpected pretty parse result: %#v", parsed[0])
	}
	if parsed[1].Severity != "warning" || parsed[1].Code != "TS6133" {
		t.Fatalf("unexpected non-pretty parse result: %#v", parsed[1])
	}

	if got := v.parseTypeScriptLine("invalid line"); got != nil {
		t.Fatalf("expected nil for invalid line, got %#v", got)
	}
}

func TestParseGoOutputAndHelpers(t *testing.T) {
	v := newValidator(t, t.TempDir())

	goOut := strings.Join([]string{
		"./internal/pkg/a.go:14: bad thing",
		"./internal/pkg/b.go:7: another thing",
		"non-matching",
	}, "\n")

	errs := v.parseGoOutput(goOut)
	if len(errs) != 2 {
		t.Fatalf("expected 2 go errors, got %d", len(errs))
	}
	if errs[0].Line != 14 || errs[1].Line != 7 {
		t.Fatalf("unexpected parsed lines: %#v", errs)
	}

	packages := v.getGoPackages([]string{
		filepath.Join(v.projectRoot, "internal", "pkg", "a.go"),
		filepath.Join(v.projectRoot, "internal", "pkg", "b.go"),
	})
	if len(packages) != 1 {
		t.Fatalf("expected deduplicated package list of size 1, got %d", len(packages))
	}

	if !containsPath("src/components/App.tsx", []string{"src/components"}) {
		t.Fatal("expected containsPath to match parent path")
	}
	if containsPath("src/components/App.tsx", []string{"internal/mcp"}) {
		t.Fatal("expected containsPath to return false for unrelated paths")
	}
}

func TestFormatErrorsForAgent(t *testing.T) {
	v := newValidator(t, t.TempDir())

	ok := v.FormatErrorsForAgent(&ValidationResult{Passed: true})
	if !strings.Contains(ok, "All validation checks passed") {
		t.Fatalf("unexpected success message: %q", ok)
	}

	result := &ValidationResult{
		Passed: false,
		Errors: []ValidationError{
			{
				File:    "src/a.ts",
				Line:    3,
				Column:  1,
				Message: "Cannot find name 'x'",
				Code:    "TS2304",
			},
		},
	}
	formatted := v.FormatErrorsForAgent(result)
	if !strings.Contains(formatted, "src/a.ts:3:1") {
		t.Fatalf("expected file location in formatted output: %q", formatted)
	}
	if !strings.Contains(formatted, "Error Code: TS2304") {
		t.Fatalf("expected error code in formatted output: %q", formatted)
	}
}
