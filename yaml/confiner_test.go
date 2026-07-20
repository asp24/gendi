package yaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfinerPreservesAddressedPathInsideBoundary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, filepath.Join(realDir, "config.yaml"), "parameters: {ok: true}")
	linkDir := filepath.Join(root, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	addressed := filepath.Join(linkDir, "config.yaml")

	got, err := (Confiner{}).Confine(root, addressed)
	if err != nil {
		t.Fatalf("confine: %v", err)
	}
	if got != addressed {
		t.Fatalf("got %q, want addressed path %q", got, addressed)
	}
}

func TestConfinerRejectsPathsOutsideBoundary(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	secret := filepath.Join(outer, "secret.yaml")
	writeTestFile(t, secret, "parameters: {secret: leaked}")
	root := filepath.Join(outer, "module")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	link := filepath.Join(root, "link.yaml")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	for _, path := range []string{secret, link} {
		if _, err := (Confiner{}).Confine(root, path); err == nil || !strings.Contains(err.Error(), "outside boundary") {
			t.Fatalf("Confine(%q): expected outside-boundary error, got %v", path, err)
		}
	}
}

func TestConfinerAllowsSymlinkedBoundary(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	realRoot := filepath.Join(outer, "real")
	if err := os.MkdirAll(realRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	config := filepath.Join(realRoot, "config.yaml")
	writeTestFile(t, config, "parameters: {ok: true}")
	linkedRoot := filepath.Join(outer, "linked")
	if err := os.Symlink(realRoot, linkedRoot); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	addressed := filepath.Join(linkedRoot, "config.yaml")
	got, err := (Confiner{}).Confine(linkedRoot, addressed)
	if err != nil {
		t.Fatalf("confine through symlinked boundary: %v", err)
	}
	if got != addressed {
		t.Fatalf("got %q, want %q", got, addressed)
	}
}

func TestConfinerRejectsNestedModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	nested := filepath.Join(root, "tools")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested module: %v", err)
	}
	writeTestFile(t, filepath.Join(nested, "go.mod"), "module example.com/tools\n")
	config := filepath.Join(nested, "gendi.yaml")
	writeTestFile(t, config, "parameters: {tool: loaded}")

	alias := filepath.Join(root, "tool.yaml")
	if err := os.Symlink(config, alias); err != nil {
		t.Fatalf("symlink nested config: %v", err)
	}

	for _, path := range []string{config, alias} {
		_, err := (Confiner{}).Confine(root, path)
		if err == nil ||
			!strings.Contains(err.Error(), "crosses Go module boundary") ||
			!strings.Contains(err.Error(), "module-path import") {
			t.Fatalf("Confine(%q): expected module-boundary error, got %v", path, err)
		}
	}
}

func TestConfinerRejectsEnteringModuleFromFallbackBoundary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "tools")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested module: %v", err)
	}
	writeTestFile(t, filepath.Join(nested, "go.mod"), "module example.com/tools\n")
	config := filepath.Join(nested, "gendi.yaml")
	writeTestFile(t, config, "parameters: {tool: loaded}")

	_, err := (Confiner{}).Confine(root, config)
	if err == nil ||
		!strings.Contains(err.Error(), "from a non-module boundary") ||
		!strings.Contains(err.Error(), "module-path import") {
		t.Fatalf("expected module-entry error, got %v", err)
	}
}

func TestConfinerRejectsEmptyBoundary(t *testing.T) {
	t.Parallel()

	if _, err := (Confiner{}).Confine("", "config.yaml"); err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Fatalf("expected empty-boundary error, got %v", err)
	}
}

func TestDefaultBoundary(t *testing.T) {
	t.Parallel()

	moduleRoot := t.TempDir()
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")
	configPath := filepath.Join(moduleRoot, "app", "gendi.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, configPath, "parameters: {ok: true}")

	got, err := DefaultBoundary(configPath)
	if err != nil {
		t.Fatalf("DefaultBoundary: %v", err)
	}
	if got != moduleRoot {
		t.Fatalf("expected module root %s, got %s", moduleRoot, got)
	}

	outside := t.TempDir()
	outsideConfig := filepath.Join(outside, "gendi.yaml")
	writeTestFile(t, outsideConfig, "parameters: {ok: true}")

	got, err = DefaultBoundary(outsideConfig)
	if err != nil {
		t.Fatalf("DefaultBoundary: %v", err)
	}
	if got != outside {
		t.Fatalf("expected config dir %s, got %s", outside, got)
	}
}
