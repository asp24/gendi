package yaml

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	yamllib "github.com/goccy/go-yaml"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/imprt"
	"github.com/asp24/gendi/srcloc"
)

// ConfigLoaderYaml loads YAML configuration files with import resolution.
type ConfigLoaderYaml struct {
	resolver imprt.Resolver
	parser   *Parser
}

type loadState struct {
	inProgress map[string]bool
	cache      map[string]*di.Config
}

// NewConfigLoaderYaml creates a new YAML config loader with dependencies.
func NewConfigLoaderYaml(resolver imprt.Resolver, parser *Parser) *ConfigLoaderYaml {
	return &ConfigLoaderYaml{
		resolver: resolver,
		parser:   parser,
	}
}

// Load loads a YAML config file with imports resolved.
func (l *ConfigLoaderYaml) Load(path string) (*di.Config, error) {
	state := &loadState{
		inProgress: map[string]bool{},
		cache:      map[string]*di.Config{},
	}
	return l.loadRecursive(path, state)
}

func (l *ConfigLoaderYaml) loadRecursive(path string, state *loadState) (*di.Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if cfg, ok := state.cache[abs]; ok {
		return cfg, nil
	}
	if state.inProgress[abs] {
		return nil, fmt.Errorf("cyclic import detected at %s", abs)
	}
	state.inProgress[abs] = true
	defer delete(state.inProgress, abs)

	data, err := l.readFile(abs)
	if err != nil {
		return nil, err
	}

	raw, err := l.parseRaw(data)
	if err != nil {
		return nil, l.toSrclocError(abs, err)
	}

	merged := di.NewConfig()

	baseDir := filepath.Dir(abs)
	for _, imp := range raw.Imports {
		impPaths, err := l.resolver.Resolve(baseDir, imp.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w", imp.Path, err)
		}

		impPaths, err = l.filterExcludedFiles(impPaths, baseDir, imp.Exclude)
		if err != nil {
			return nil, fmt.Errorf("apply exclusions for import %q: %w", imp.Path, err)
		}

		for _, impPath := range impPaths {
			child, err := l.loadRecursive(impPath, state)
			if err != nil {
				return nil, err
			}
			merged = merged.MergeWith(child)
		}
	}

	cfg, err := l.parser.ConvertConfigWithDirAndFile(raw, baseDir, abs)
	if err != nil {
		return nil, srcloc.AddContext(err, "convert %s", abs)
	}

	result := merged.MergeWith(cfg)
	state.cache[abs] = result
	return result, nil
}

// parseRaw parses YAML data into raw config (with imports).
func (l *ConfigLoaderYaml) parseRaw(data []byte) (*RawConfig, error) {
	var raw RawConfig
	if err := l.yamlUnmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

func (l *ConfigLoaderYaml) readFile(path string) ([]byte, error) {
	return l.osReadFile(path)
}

// osReadFile can be replaced for testing.
var defaultOsReadFile = os.ReadFile

func (l *ConfigLoaderYaml) osReadFile(path string) ([]byte, error) {
	return defaultOsReadFile(path)
}

// yamlUnmarshal wraps yaml.Unmarshal for testability.
var defaultYamlUnmarshal = yamllib.Unmarshal

func (l *ConfigLoaderYaml) yamlUnmarshal(data []byte, v interface{}) error {
	return defaultYamlUnmarshal(data, v)
}

// toSrclocError converts errors from parseRaw into *srcloc.Error, so
// the existing srcloc.Renderer can render snippet + caret.
//
// Three buckets:
//
//  1. *NodeError — our DTO-level validation. Pre-existing.
//  2. yaml.Error (interface) — goccy syntax errors and decode errors
//     (type mismatch, integer overflow, duplicate key, unknown field,
//     unexpected node). All implement GetToken / GetMessage.
//  3. anything else — fallback via srcloc.AddContext (does not
//     double-locate if the residual error is itself *srcloc.Error).
func (l *ConfigLoaderYaml) toSrclocError(file string, err error) error {
	var ne *NodeError
	if errors.As(err, &ne) {
		loc := newLocation(file, ne.Node)

		// ne.Err may itself wrap a goccy yaml.Error (e.g. NodeToValue
		// failed inside a custom UnmarshalYAML). srcloc.WrapError would
		// call ne.Err.Error(), which routes through goccy's own
		// formatter and breaks the single-style invariant. Normalize:
		// strip the wrapped yaml.Error down to its plain message.
		wrapped := ne.Err
		var inner yamllib.Error
		if errors.As(wrapped, &inner) {
			wrapped = errors.New(inner.GetMessage())
		}
		return srcloc.WrapError(loc, ne.Msg, wrapped)
	}

	var ye yamllib.Error
	if errors.As(err, &ye) {
		var loc *srcloc.Location
		if tok := ye.GetToken(); tok != nil && tok.Position != nil {
			loc = srcloc.NewLocation(file, tok.Position.Line, tok.Position.Column)
		}
		return srcloc.Errorf(loc, "%s", ye.GetMessage())
	}

	return srcloc.AddContext(err, "parse %s", file)
}

// filterExcludedFiles removes files matching any exclusion pattern.
// files - absolute paths returned by resolver
// baseDir - directory for resolving relative exclusion patterns
// excludePatterns - glob patterns, file paths, or directory paths to exclude
func (l *ConfigLoaderYaml) filterExcludedFiles(files []string, baseDir string, excludePatterns []string) ([]string, error) {
	if len(excludePatterns) == 0 {
		return files, nil
	}

	excludedSet := make(map[string]bool)
	var excludedDirs []string

	for _, pattern := range excludePatterns {
		// Resolve pattern relative to baseDir
		absPattern := pattern
		if !filepath.IsAbs(pattern) {
			absPattern = filepath.Join(baseDir, pattern)
		}

		// If the pattern is a directory, exclude all files under it
		if info, err := os.Stat(absPattern); err == nil && info.IsDir() {
			excludedDirs = append(excludedDirs, absPattern+string(filepath.Separator))
			continue
		}

		// Match pattern using doublestar (same library as glob imports)
		matches, err := doublestar.FilepathGlob(absPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			abs, err := filepath.Abs(match)
			if err != nil {
				continue
			}
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				excludedDirs = append(excludedDirs, abs+string(filepath.Separator))
			} else {
				excludedSet[abs] = true
			}
		}
	}

	// Filter files not in exclusion set and not under excluded directories
	result := make([]string, 0, len(files))
	for _, file := range files {
		if excludedSet[file] {
			continue
		}
		excluded := false
		for _, dir := range excludedDirs {
			if strings.HasPrefix(file, dir) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, file)
		}
	}

	return result, nil
}
