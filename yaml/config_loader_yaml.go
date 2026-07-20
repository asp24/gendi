package yaml

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	yamllib "github.com/goccy/go-yaml"

	di "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/srcloc"
)

// ImportResolver resolves one import entry — path plus its exclusion masks —
// to config files, confining every result to its sandbox (see imprt.Resolver).
type ImportResolver interface {
	// ResolveImport resolves an import path to config files, dropping the
	// files matched by the exclusion masks before the sandbox check.
	ResolveImport(baseDir, path string, excludes []string) ([]string, error)
}

// ConfigLoaderYaml loads YAML configuration files with import resolution.
type ConfigLoaderYaml struct {
	resolver ImportResolver
	parser   *Parser
}

type loadState struct {
	inProgress map[string]bool
	cache      map[string]*di.Config
}

// NewConfigLoaderYaml creates a new YAML config loader with dependencies. A
// loader is not safe for concurrent use — the underlying resolver memoizes
// module lookups without locking; use one loader per goroutine.
func NewConfigLoaderYaml(resolver ImportResolver, parser *Parser) *ConfigLoaderYaml {
	return &ConfigLoaderYaml{
		resolver: resolver,
		parser:   parser,
	}
}

// Load loads a YAML config file with imports resolved. Confinement to the Go
// module of each importing file is enforced by the resolver chain itself.
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
	// Identity is canonical: caching and cycle detection must see one entry
	// per real file, whatever spelling reached it. Everything else — anchoring
	// of nested imports, $this, error messages — stays on the spelled path. A
	// file that cannot be resolved fails loudly at readFile below.
	id := abs
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		id = real
	}
	if cfg, ok := state.cache[id]; ok {
		return cfg, nil
	}
	if state.inProgress[id] {
		return nil, fmt.Errorf("cyclic import detected at %s", abs)
	}
	state.inProgress[id] = true
	defer delete(state.inProgress, id)

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
		impPaths, err := l.resolver.ResolveImport(baseDir, imp.Path, imp.Exclude)
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w", imp.Path, err)
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
	state.cache[id] = result
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
