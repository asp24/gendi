package di

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// Pass is a compiler pass that can mutate config before validation and generation.
type Pass interface {
	Name() string
	Process(cfg *Config) error
}

// Config is the root configuration for the DI container.
type Config struct {
	Imports    []Import             `yaml:"imports"`
	Parameters map[string]Parameter `yaml:"parameters"`
	Tags       map[string]Tag       `yaml:"tags"`
	Services   map[string]*Service  `yaml:"services"`
}

// Import defines an import entry with an optional service prefix.
type Import struct {
	Path   string `yaml:"path"`
	Prefix string `yaml:"prefix"`
}

// UnmarshalYAML allows import entries to be strings or mappings.
func (i *Import) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var path string
		if err := node.Decode(&path); err != nil {
			return err
		}
		i.Path = path
		i.Prefix = ""
		return nil
	case yaml.MappingNode:
		type alias Import
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		if decoded.Path == "" {
			return fmt.Errorf("import path is required")
		}
		*i = Import(decoded)
		return nil
	default:
		return fmt.Errorf("import must be a string or mapping")
	}
}

// Parameter defines a typed parameter literal.
type Parameter struct {
	Type  string    `yaml:"type"`
	Value yaml.Node `yaml:"value"`
}

// Tag defines a tag declaration.
type Tag struct {
	ElementType string `yaml:"element_type"`
	SortBy      string `yaml:"sort_by"`
}

// ServiceTag defines a tag assigned to a service.
type ServiceTag struct {
	Name       string                 `yaml:"name"`
	Attributes map[string]interface{} `yaml:"attributes"`
}

// Service defines a service entry.
type Service struct {
	Type               string       `yaml:"type"`
	Constructor        Constructor  `yaml:"constructor"`
	Shared             *bool        `yaml:"shared"`
	Public             bool         `yaml:"public,omitempty"`
	Decorates          string       `yaml:"decorates"`
	DecorationPriority int          `yaml:"decoration_priority"`
	Tags               []ServiceTag `yaml:"tags"`
	Alias              string       `yaml:"alias"`
}

// Constructor defines service constructor configuration.
type Constructor struct {
	Func   string     `yaml:"func"`
	Method string     `yaml:"method"`
	Args   []Argument `yaml:"args"`
}

// ArgumentKind is the parsed kind of a constructor argument.
type ArgumentKind int

const (
	ArgLiteral ArgumentKind = iota
	ArgServiceRef
	ArgInner
	ArgParam
	ArgTagged
)

// Argument represents a constructor argument.
type Argument struct {
	Kind    ArgumentKind
	Value   string
	Literal yaml.Node
}

// UnmarshalYAML parses argument syntax.
func (a *Argument) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("argument must be a scalar")
	}
	if node.Tag == "!!str" {
		s := node.Value
		switch {
		case s == "@.inner":
			a.Kind = ArgInner
			a.Value = s
			return nil
		case len(s) > 1 && s[0] == '@':
			a.Kind = ArgServiceRef
			a.Value = s[1:]
			return nil
		case len(s) > 2 && s[0] == '%' && s[len(s)-1] == '%':
			a.Kind = ArgParam
			a.Value = s[1 : len(s)-1]
			return nil
		case len(s) > len("!tagged:") && s[:len("!tagged:")] == "!tagged:":
			a.Kind = ArgTagged
			a.Value = s[len("!tagged:"):]
			return nil
		}
	}

	a.Kind = ArgLiteral
	a.Literal = *node
	return nil
}

// UnmarshalYAML allows service entries to be aliases or mappings.
func (s *Service) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var ref string
		if err := node.Decode(&ref); err != nil {
			return err
		}
		if !strings.HasPrefix(ref, "@") || len(ref) == 1 {
			return fmt.Errorf("service alias must start with @")
		}
		*s = Service{Alias: ref[1:]}
		return nil
	case yaml.MappingNode:
		type alias Service
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*s = Service(decoded)
		return nil
	default:
		return fmt.Errorf("service must be a mapping or alias")
	}
}

// LoadConfig loads config with imports resolved.
func LoadConfig(path string) (*Config, error) {
	visited := map[string]bool{}
	return loadConfig(path, visited)
}

func loadConfig(path string, visited map[string]bool) (*Config, error) {
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

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}

	merged := &Config{
		Parameters: map[string]Parameter{},
		Tags:       map[string]Tag{},
		Services:   map[string]*Service{},
	}

	baseDir := filepath.Dir(abs)
	for _, imp := range cfg.Imports {
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

	cfg.Imports = nil
	merged = mergeConfig(merged, cfg)
	return merged, nil
}

func mergeConfig(dst, src *Config) *Config {
	if dst.Parameters == nil {
		dst.Parameters = map[string]Parameter{}
	}
	if dst.Tags == nil {
		dst.Tags = map[string]Tag{}
	}
	if dst.Services == nil {
		dst.Services = map[string]*Service{}
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

func applyServicePrefix(cfg *Config, prefix string) {
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
			if arg.Kind == ArgServiceRef && original[arg.Value] {
				arg.Value = prefix + arg.Value
			}
		}
	}
	prefixed := map[string]*Service{}
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
	if !looksLikeModulePath(importPath) {
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

func looksLikeModulePath(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}
	return strings.Contains(parts[0], ".")
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func resolveGlobImportPaths(baseDir, importPath string) ([]string, error) {
	if filepath.IsAbs(importPath) {
		return globFiles(importPath)
	}
	if isExplicitRelative(importPath) || !looksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		return globFiles(pattern)
	}
	return resolveModuleImportGlob(baseDir, importPath)
}

func resolveModuleImport(baseDir, importPath string) (string, error) {
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if hasGlob(candidate) {
			continue
		}
		moduleDir, err := goModuleDir(baseDir, candidate)
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
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if hasGlob(candidate) {
			continue
		}
		moduleDir, err := goModuleDir(baseDir, candidate)
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

func goModuleDir(baseDir, modulePath string) (string, error) {
	if dir, ok := localModuleDir(baseDir, modulePath); ok {
		return dir, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if dir, ok := localModuleDir(cwd, modulePath); ok {
			return dir, nil
		}
	}

	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", modulePath)
	cmd.Dir = baseDir
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

func localModuleDir(startDir, modulePath string) (string, bool) {
	dir, modPath, ok := findModuleRoot(startDir)
	if !ok {
		return "", false
	}
	if modPath != modulePath {
		return "", false
	}
	return dir, true
}

func findModuleRoot(startDir string) (string, string, bool) {
	dir := startDir
	for {
		modFile := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modFile); err == nil {
			if modPath := parseModulePath(data); modPath != "" {
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

func parseModulePath(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
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

// osReadFile is defined in config_os.go to ease testing.
var osReadFile = readFileDefault

func readFileDefault(path string) ([]byte, error) {
	return os.ReadFile(path)
}
