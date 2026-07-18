package generator

import (
	"bytes"
	"strings"
	"testing"
)

func TestAssembleOutputBuildTagsHeader(t *testing.T) {
	g := NewGenerator(NewIdentGenerator())

	out := string(g.assembleOutput(
		Options{Package: "di", Container: "Container", BuildTags: "foo && bar"},
		NewImportManager("example.com/app/di"),
		&bytes.Buffer{},
	))

	if !strings.Contains(out, "//go:build foo && bar\n") {
		t.Fatalf("expected //go:build header, got:\n%s", out)
	}
	if strings.Contains(out, "// +build") {
		t.Fatalf("unexpected legacy // +build line, got:\n%s", out)
	}
}

func TestAssembleOutputWithoutBuildTags(t *testing.T) {
	g := NewGenerator(NewIdentGenerator())

	out := string(g.assembleOutput(
		Options{Package: "di", Container: "Container"},
		NewImportManager("example.com/app/di"),
		&bytes.Buffer{},
	))

	if strings.Contains(out, "//go:build") || strings.Contains(out, "// +build") {
		t.Fatalf("unexpected build constraint header, got:\n%s", out)
	}
}
