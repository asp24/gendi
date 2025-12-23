// Package gomod provides utilities for locating Go modules.
package gomod

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Locator finds Go module directories by module path.
type Locator struct {
	// BaseDir is the directory to start searching from.
	BaseDir string
}

// NewLocator creates a new module locator with the given base directory.
func NewLocator(baseDir string) *Locator {
	return &Locator{BaseDir: baseDir}
}

// FindModuleDir returns the directory for the given module path.
// It first checks if the module is the local module (in BaseDir or cwd),
// then falls back to using "go list" to find the module.
func (l *Locator) FindModuleDir(modulePath string) (string, error) {
	if dir, ok := l.findLocalModuleDir(modulePath); ok {
		return dir, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if dir, ok := findLocalModuleDirFrom(cwd, modulePath); ok {
			return dir, nil
		}
	}
	return l.findModuleDirViaGoList(modulePath)
}

// findLocalModuleDir checks if modulePath matches the local module.
func (l *Locator) findLocalModuleDir(modulePath string) (string, bool) {
	return findLocalModuleDirFrom(l.BaseDir, modulePath)
}

func findLocalModuleDirFrom(startDir, modulePath string) (string, bool) {
	dir, modPath, ok := FindModuleRoot(startDir)
	if !ok {
		return "", false
	}
	if modPath != modulePath {
		return "", false
	}
	return dir, true
}

// findModuleDirViaGoList uses "go list" to find the module directory.
func (l *Locator) findModuleDirViaGoList(modulePath string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", modulePath)
	cmd.Dir = l.BaseDir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(exitErr.Stderr))
			if msg == "" {
				return "", err
			}
			return "", fmt.Errorf("%s", msg)
		}
		return "", err
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("go list returned empty module dir for %s", modulePath)
	}
	return dir, nil
}

// FindModuleRoot searches upward from startDir to find a go.mod file.
// Returns the directory containing go.mod, the module path, and whether it was found.
func FindModuleRoot(startDir string) (dir string, modulePath string, found bool) {
	dir = startDir
	for {
		modFile := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modFile); err == nil {
			if modPath := ParseModulePath(data); modPath != "" {
				return dir, modPath, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false
		}
		dir = parent
	}
}

// ParseModulePath extracts the module path from go.mod content.
func ParseModulePath(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// LooksLikeModulePath returns true if path appears to be a Go module path
// (i.e., first segment contains a dot, like "github.com/...").
func LooksLikeModulePath(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}
	return strings.Contains(parts[0], ".")
}
