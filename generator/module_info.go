package generator

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ModuleInfo contains information about the Go module.
type ModuleInfo struct {
	Path string // Module import path (e.g., "github.com/user/repo")
	Root string // Absolute path to module root directory
}

// ResolveModuleInfo finds the go.mod file and extracts module information.
// It searches from the current working directory upwards.
func ResolveModuleInfo() (ModuleInfo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return ModuleInfo{}, err
	}

	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return ModuleInfo{}, errors.New("go.mod not found")
		}
		root = parent
	}

	modPath, err := parseModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return ModuleInfo{}, err
	}

	return ModuleInfo{
		Path: modPath,
		Root: root,
	}, nil
}

// parseModulePath extracts the module path from a go.mod file.
func parseModulePath(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("module path not found in go.mod")
}
