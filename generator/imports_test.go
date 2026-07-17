package generator

import (
	"go/types"
	"strings"
	"testing"
)

func TestRenderImportsReservedPathIsNotDuplicated(t *testing.T) {
	m := NewImportManager("example.com/output")
	m.ReserveImport("sync", "sync")
	m.Require("sync")

	// A constructor signature referencing stdlib sync resolves it through
	// the qualifier; the reserved alias must be reused, not sync2.
	syncPkg := types.NewPackage("sync", "sync")
	if got := m.qualifier(syncPkg); got != "sync" {
		t.Fatalf("qualifier(sync) = %q, want %q", got, "sync")
	}

	rendered := m.renderImports()
	if got := strings.Count(rendered, "\"sync\""); got != 1 {
		t.Fatalf("import path \"sync\" rendered %d times, want 1:\n%s", got, rendered)
	}
	if strings.Contains(rendered, "sync2") {
		t.Fatalf("reserved package must not get a numbered alias:\n%s", rendered)
	}
	if strings.Contains(rendered, "sync \"sync\"") {
		t.Fatalf("reserved import must render without redundant alias:\n%s", rendered)
	}
}

func TestRenderImportsUserPackageNamedSyncGetsNumberedAlias(t *testing.T) {
	m := NewImportManager("example.com/output")
	m.ReserveImport("sync", "sync")
	m.Require("sync")

	userSync := types.NewPackage("example.com/app/sync", "sync")
	if got := m.qualifier(userSync); got != "sync2" {
		t.Fatalf("qualifier(user sync) = %q, want %q", got, "sync2")
	}

	rendered := m.renderImports()
	if !strings.Contains(rendered, "\t\"sync\"\n") {
		t.Fatalf("required stdlib sync import missing:\n%s", rendered)
	}
	if !strings.Contains(rendered, "sync2 \"example.com/app/sync\"") {
		t.Fatalf("user package must keep its numbered alias:\n%s", rendered)
	}
}

func TestRenderImportsRequiredSkippedWhenAlreadyReservedAndQualified(t *testing.T) {
	m := NewImportManager("example.com/output")
	m.ReserveImport("fmt", "fmt")

	fmtPkg := types.NewPackage("fmt", "fmt")
	if got := m.qualifier(fmtPkg); got != "fmt" {
		t.Fatalf("qualifier(fmt) = %q, want %q", got, "fmt")
	}
	m.Require("fmt")

	rendered := m.renderImports()
	if got := strings.Count(rendered, "\"fmt\""); got != 1 {
		t.Fatalf("import path \"fmt\" rendered %d times, want 1:\n%s", got, rendered)
	}
}
