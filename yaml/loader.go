package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/gomod"
)

// LoadConfig loads a YAML config file with imports resolved.
func LoadConfig(path string) (*di.Config, error) {
	visited := map[string]bool{}
	return loadConfig(path, visited)
}

func loadConfig(path string, visited map[string]bool) (*di.Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[abs] {
		return nil, fmt.Errorf("cyclic import detected at %s", abs)
	}
	visited[abs] = true

	data, err := readFile(abs)
	if err != nil {
		return nil, err
	}

	raw, err := parseRaw(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}

	merged := &di.Config{
		Parameters: map[string]di.Parameter{},
		Tags:       map[string]di.Tag{},
		Services:   map[string]*di.Service{},
	}

	baseDir := filepath.Dir(abs)
	for _, imp := range raw.Imports {
		impPaths, err := resolveImportPaths(baseDir, imp.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w", imp.Path, err)
		}
		for _, impPath := range impPaths {
			child, err := loadConfig(impPath, visited)
			if err != nil {
				return nil, err
			}
			if imp.Prefix != "" {
				applyServicePrefix(child, imp.Prefix)
			}
			merged = mergeConfig(merged, child)
		}
	}

	cfg, err := convertConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", abs, err)
	}
	merged = mergeConfig(merged, cfg)
	return merged, nil
}

// parseRaw parses YAML data into raw config (with imports).
func parseRaw(data []byte) (*rawConfig, error) {
	var raw rawConfig
	if err := yamlUnmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

func mergeConfig(dst, src *di.Config) *di.Config {
	if dst.Parameters == nil {
		dst.Parameters = map[string]di.Parameter{}
	}
	if dst.Tags == nil {
		dst.Tags = map[string]di.Tag{}
	}
	if dst.Services == nil {
		dst.Services = map[string]*di.Service{}
	}

	for k, v := range src.Parameters {
		dst.Parameters[k] = v
	}
	for k, v := range src.Tags {
		dst.Tags[k] = v
	}
	for k, v := range src.Services {
		copySvc := *v
		dst.Services[k] = &copySvc
	}
	return dst
}

func applyServicePrefix(cfg *di.Config, prefix string) {
	if prefix == "" || len(cfg.Services) == 0 {
		return
	}
	original := map[string]bool{}
	for name := range cfg.Services {
		original[name] = true
	}
	for _, svc := range cfg.Services {
		if svc.Decorates != "" && original[svc.Decorates] {
			svc.Decorates = prefix + svc.Decorates
		}
		if svc.Alias != "" && original[svc.Alias] {
			svc.Alias = prefix + svc.Alias
		}
		for i := range svc.Constructor.Args {
			arg := &svc.Constructor.Args[i]
			if arg.Kind == di.ArgServiceRef && original[arg.Value] {
				arg.Value = prefix + arg.Value
			}
		}
	}
	prefixed := map[string]*di.Service{}
	for name, svc := range cfg.Services {
		prefixed[prefix+name] = svc
	}
	cfg.Services = prefixed
}

func resolveImportPaths(baseDir, importPath string) ([]string, error) {
	if importPath == "" {
		return nil, fmt.Errorf("import path is empty")
	}
	if hasGlob(importPath) {
		return resolveGlobImportPaths(baseDir, importPath)
	}
	if filepath.IsAbs(importPath) {
		path, err := ensureFile(importPath)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	localPath := filepath.Join(baseDir, importPath)
	if exists(localPath) {
		path, err := filepath.Abs(localPath)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	if isExplicitRelative(importPath) {
		return nil, fmt.Errorf("import not found at %s", localPath)
	}
	if !gomod.LooksLikeModulePath(importPath) {
		return nil, fmt.Errorf("import not found at %s", localPath)
	}
	path, err := resolveModuleImport(baseDir, importPath)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func ensureFile(path string) (string, error) {
	if exists(path) {
		return filepath.Abs(path)
	}
	return "", fmt.Errorf("import not found at %s", path)
}

func exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func isExplicitRelative(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func resolveGlobImportPaths(baseDir, importPath string) ([]string, error) {
	if filepath.IsAbs(importPath) {
		return globFiles(importPath)
	}
	if isExplicitRelative(importPath) || !gomod.LooksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		return globFiles(pattern)
	}
	return resolveModuleImportGlob(baseDir, importPath)
}

func resolveModuleImport(baseDir, importPath string) (string, error) {
	locator := gomod.NewLocator(baseDir)
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if hasGlob(candidate) {
			continue
		}
		moduleDir, err := locator.FindModuleDir(candidate)
		if err != nil {
			continue
		}
		remainder := strings.Join(parts[i:], "/")
		if remainder == "" {
			if path, ok := findDefaultConfig(moduleDir); ok {
				return path, nil
			}
			return "", fmt.Errorf("module %s has no gendi.yaml", candidate)
		}
		full := filepath.Join(moduleDir, filepath.FromSlash(remainder))
		if exists(full) {
			return filepath.Abs(full)
		}
		return "", fmt.Errorf("module %s does not contain %s", candidate, remainder)
	}
	return "", fmt.Errorf("module %s not found", importPath)
}

func resolveModuleImportGlob(baseDir, importPath string) ([]string, error) {
	locator := gomod.NewLocator(baseDir)
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if hasGlob(candidate) {
			continue
		}
		moduleDir, err := locator.FindModuleDir(candidate)
		if err != nil {
			continue
		}
		remainder := strings.Join(parts[i:], "/")
		if remainder == "" {
			path, ok := findDefaultConfig(moduleDir)
			if !ok {
				return nil, fmt.Errorf("module %s has no gendi.yaml", candidate)
			}
			return []string{path}, nil
		}
		pattern := filepath.Join(moduleDir, filepath.FromSlash(remainder))
		return globFiles(pattern)
	}
	return nil, fmt.Errorf("module %s not found", importPath)
}

func findDefaultConfig(moduleDir string) (string, bool) {
	path := filepath.Join(moduleDir, "gendi.yaml")
	if exists(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs, true
		}
		return path, true
	}
	path = filepath.Join(moduleDir, "gendi.yml")
	if exists(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs, true
		}
		return path, true
	}
	return "", false
}

func globFiles(pattern string) ([]string, error) {
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		if exists(match) {
			abs, err := filepath.Abs(match)
			if err != nil {
				return nil, err
			}
			files = append(files, abs)
		}
	}
	if len(files) != 0 {
		sort.Strings(files)
	}

	return files, nil
}

func readFile(path string) ([]byte, error) {
	return osReadFile(path)
}

// osReadFile can be replaced for testing.
var osReadFile = os.ReadFile

// yamlUnmarshal wraps yaml.Unmarshal for testability.
var yamlUnmarshal = yamlUnmarshalDefault

func yamlUnmarshalDefault(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
