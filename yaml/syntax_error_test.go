package yaml

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asp24/gendi/srcloc"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gendi.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

func TestSyntaxError_BadIndent_ProducesSrclocError(t *testing.T) {
	// Indentation under "services" is broken: "foo" and "baz" disagree.
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
	absPath, _ := filepath.Abs(path)
	if le.Loc.File != absPath {
		t.Errorf("Loc.File = %q, want %q", le.Loc.File, absPath)
	}
	if le.Loc.Line < 1 {
		t.Errorf("Loc.Line = %d, want >= 1", le.Loc.Line)
	}
}

// Spec §Testing #1: exact line/column assertion (catches off-by-one
// regressions when goccy version changes). Values were confirmed by
// running against goccy v1.19.2.
func TestSyntaxError_BadIndent_ExactLocation(t *testing.T) {
	// Carefully-controlled YAML: the second mapping key under "services"
	// has one extra space of indent. goccy reports the error at the
	// second ':' (line 2, column 7) where the inconsistent indent is
	// detected.
	//
	// Line 1: services:
	// Line 2:  foo: bar       (1-space indent — error reported here)
	// Line 3:   baz: qux      (2-space indent — the conflict)
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T", err)
	}
	if le.Loc == nil {
		t.Fatal("nil Loc")
	}
	// Locked-in for goccy v1.19.2; bump deliberately if the version pin
	// in go.mod changes.
	const wantLine, wantCol = 2, 7
	if le.Loc.Line != wantLine || le.Loc.Column != wantCol {
		t.Errorf("Loc = %d:%d, want %d:%d", le.Loc.Line, le.Loc.Column, wantLine, wantCol)
	}
}

func TestSyntaxError_UnclosedQuote_ProducesSrclocError(t *testing.T) {
	yaml := `parameters:
  bad:
    type: string
    value: "unclosed
`
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
}

func TestSyntaxError_TypeMismatch_ProducesSrclocError(t *testing.T) {
	// decoration_priority is int, not string. Triggers a yaml.Error
	// that is NOT a *yaml.SyntaxError (decode error path).
	yaml := `services:
  foo:
    type: string
    constructor:
      func: pkg.New
    decoration_priority: "abc"
`
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
}

func TestSyntaxError_RenderingProducesSnippet(t *testing.T) {
	// End-to-end: feed a known-bad YAML, render the error, expect a
	// caret-style snippet output.
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	rendered := srcloc.NewRenderer().RenderError(err, 2)
	if !strings.Contains(rendered, "^") {
		t.Errorf("expected caret in rendered output, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "gendi.yaml:") {
		t.Errorf("expected file:line:col prefix, got:\n%s", rendered)
	}
}

func TestSyntaxError_EmptyFile_DoesNotPanic(t *testing.T) {
	path := writeYAML(t, "")
	// Empty file should either return nil config (if goccy treats it as
	// nil doc) or a srcloc.Error without panic. The point is: no panic.
	_, _ = LoadConfig(path)
}
