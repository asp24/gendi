package yaml

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yamllib "github.com/goccy/go-yaml"

	di "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/gomod"
	"github.com/gendi-org/gendi/imprt"
	"github.com/gendi-org/gendi/srcloc"
)

// ConfigLoaderYaml loads YAML configuration files with import resolution.
type ConfigLoaderYaml struct {
	resolver imprt.Resolver
	parser   *Parser
}

type loadState struct {
	inProgress map[string]bool
	cache      map[string]*di.Config
	// resolver is the base resolver wrapped in a sandbox anchored at the root
	// config's module, so every import and exclusion is confined to it.
	resolver imprt.Resolver
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
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Confine resolution to the root config's Go module. Configs outside any
	// module fall back to the root config's own directory as the boundary.
	fallbackRoot := filepath.Dir(abs)
	if root, _, found := gomod.FindModuleRoot(fallbackRoot); found {
		fallbackRoot = root
	}

	state := &loadState{
		inProgress: map[string]bool{},
		cache:      map[string]*di.Config{},
		resolver:   imprt.NewResolverSandbox(l.resolver, fallbackRoot),
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
		impPaths, err := state.resolver.Resolve(baseDir, imp.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w", imp.Path, err)
		}

		impPaths, err = l.filterExcludedFiles(impPaths, baseDir, imp.Exclude, state.resolver)
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

func (l *ConfigLoaderYaml) yamlUnmarshal(data []byte, v any) error {
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
//
// Exclusion patterns are addressed exactly like an import `path`: a relative
// pattern is resolved against the importing file's directory, an absolute
// pattern against the filesystem root, and a module pattern against the Go
// module it names. This mirroring means an exclusion is simply "one of the
// things the import could have matched, written the same way" — for an
// absolute import you exclude with an absolute pattern, for a module import
// with a module pattern. A pattern that points at a directory excludes its
// whole subtree.
//
// files - absolute paths returned by the resolver
// baseDir - the importing file's directory (anchor for relative patterns)
// excludePatterns - glob patterns, file paths, module paths, or directories
// resolver - the same (sandboxed) resolver used for the import path
func (l *ConfigLoaderYaml) filterExcludedFiles(files []string, baseDir string, excludePatterns []string, resolver imprt.Resolver) ([]string, error) {
	if len(excludePatterns) == 0 {
		return files, nil
	}

	excludedSet := make(map[string]bool)
	var excludedDirs []string

	for _, pattern := range excludePatterns {
		// A pattern pointing at an existing directory excludes its subtree.
		// Directories are not config files, so they never reach the resolver.
		if dir, ok := l.excludedDirectory(baseDir, pattern); ok {
			excludedDirs = append(excludedDirs, dir+string(filepath.Separator))
			continue
		}

		// Otherwise resolve the pattern through the same resolver as the
		// import `path`, and exclude every file it matches. A glob that
		// matches nothing is a silent no-op (as for imports); a concrete
		// pattern that resolves to nothing is a loud error.
		matches, err := resolver.Resolve(baseDir, pattern)
		if err != nil {
			return nil, fmt.Errorf("exclusion %q: %w", pattern, err)
		}
		for _, match := range matches {
			excludedSet[match] = true
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

// excludedDirectory reports whether pattern points at an existing directory,
// resolved relative to the importing file's directory (the same addressing as
// an import path). Absolute patterns, glob patterns, and files never match
// and fall through to the (sandboxed) resolver, which rejects absolute paths.
func (l *ConfigLoaderYaml) excludedDirectory(baseDir, pattern string) (string, bool) {
	if filepath.IsAbs(pattern) {
		return "", false
	}
	candidate := filepath.Join(baseDir, pattern)
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return "", false
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	return abs, true
}
