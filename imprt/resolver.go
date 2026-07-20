package imprt

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/gendi-org/gendi/gomod"
)

// kind classifies an import or exclusion pattern by its syntactic form.
type kind int

const (
	// kindLocal is a path resolved against the importing file's directory.
	kindLocal kind = iota
	// kindModule is a path resolved against the Go module it names.
	kindModule
)

// classify reports the addressing form of pattern, purely syntactically —
// no filesystem access. A pattern names a Go module only when it is
// multi-segment and its first segment contains a dot; everything else is a
// local path resolved against the importing file. Single-segment patterns
// (base.yaml, test_*.yaml, ".", "..") are always local — a module import
// must name a file inside the module, so it always contains a slash. Empty
// and absolute paths are errors.
func classify(pattern string) (kind, error) {
	if pattern == "" {
		return 0, fmt.Errorf("import path is empty")
	}
	if filepath.IsAbs(pattern) {
		return 0, fmt.Errorf("absolute paths are not allowed; use a path relative to the importing file or a Go module path")
	}
	if strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "../") {
		return kindLocal, nil
	}
	if strings.Contains(pattern, "/") && gomod.LooksLikeModulePath(pattern) {
		return kindModule, nil
	}
	return kindLocal, nil
}

// Candidate is a config file found by import resolution. Path preserves the
// addressed spelling, while Boundary is the root within which the YAML loader
// must confine the file before reading it.
type Candidate struct {
	Path     string
	Boundary string
}

// Resolver resolves import entries to config file candidates through one fixed
// pipeline: classify the import (local directory or Go module), compute its
// anchor and confinement boundary, find the files the import mask matches, and
// drop the ones matched by the exclusion masks. The YAML loader performs the
// final confinement check immediately before loading each candidate.
type Resolver struct {
	// boundary is the absolute path outside which loading is forbidden when
	// the importing file is not inside any Go module; within a module, that
	// module's root is the boundary instead.
	boundary string
	// moduleDirs memoizes module resolution per (module context, modulePath),
	// failures included: each cold lookup can cost a `go list` subprocess,
	// and an unresolvable candidate prefix would otherwise re-run it for
	// every import entry sharing that prefix. Results depend only on the
	// context directory's go.mod graph, never on the process working
	// directory. Loading is single-threaded, so no locking is needed.
	moduleDirs map[string]moduleLookup
}

// moduleLookup is a memoized module resolution: the module directory when
// resolution succeeded, or ok=false when the candidate is not a module.
type moduleLookup struct {
	dir string
	ok  bool
}

// NewResolver creates a Resolver whose out-of-module confinement boundary is
// boundary. An empty boundary is an error: it would silently degrade to the
// process working directory.
//
// A Resolver memoizes module lookups without locking and is not safe for
// concurrent use: to resolve imports in parallel, give each goroutine its
// own Resolver.
func NewResolver(boundary string) (*Resolver, error) {
	if boundary == "" {
		return nil, fmt.Errorf("boundary must not be empty")
	}
	abs, err := filepath.Abs(boundary)
	if err != nil {
		return nil, err
	}
	return &Resolver{boundary: abs, moduleDirs: map[string]moduleLookup{}}, nil
}

// target is an addressed import: what to resolve, against which directory,
// and which root the resolved files may not escape.
type target struct {
	kind       kind
	anchorDir  string // directory the pattern resolves against
	boundary   string // confinement root for resolved files
	pattern    string // for modules, the remainder after the module path
	modulePath string // set for kindModule, used for exclusion masks and errors
}

// address classifies pattern and computes its resolution anchor. Local
// patterns resolve against the importing file's directory and are confined to
// its module; module patterns resolve against — and are confined to — the
// resolved module's directory.
func (r *Resolver) address(baseDir, pattern string) (target, error) {
	k, err := classify(pattern)
	if err != nil {
		return target{}, err
	}
	if k == kindLocal {
		return target{
			kind:      kindLocal,
			anchorDir: baseDir,
			boundary:  moduleRootOf(baseDir, r.boundary),
			pattern:   pattern,
		}, nil
	}

	moduleDir, modulePath, remainder, err := r.findModule(baseDir, pattern)
	if err != nil {
		// Only suggest the local spelling when it would actually resolve —
		// otherwise the hint sends the user chasing a path that does not
		// exist, on top of an unrelated failure (no go.mod, typo'd module).
		if localMatch(baseDir, pattern) {
			return target{}, fmt.Errorf("%w (for a local directory use %q)", err, "./"+pattern)
		}
		return target{}, err
	}
	if remainder == "" {
		return target{}, fmt.Errorf("module import %q must reference a file, e.g. %s/gendi.yaml", pattern, modulePath)
	}
	// A module-shaped spelling that also matches something relative to the
	// importing file is ambiguous: picking either side would silently shadow
	// the other.
	if localMatch(baseDir, pattern) {
		return target{}, fmt.Errorf("import %q is ambiguous: it resolves in module %s but the same spelling exists locally — use %q for the local path, or remove the local one to import from the module", pattern, modulePath, "./"+pattern)
	}
	return target{
		kind:       kindModule,
		anchorDir:  pathToAbs(moduleDir),
		boundary:   pathToAbs(moduleDir),
		pattern:    remainder,
		modulePath: modulePath,
	}, nil
}

// ResolveImport resolves an import entry to config file candidates. Paths are
// absolute and preserve their addressed spelling, so every symlink alias keeps
// its own relative-import and $this context. A literal path must name an
// existing file; a glob that matches nothing resolves to no files. Files
// matched by any exclusion mask are dropped before candidates reach the YAML
// loader and its sandbox check.
func (r *Resolver) ResolveImport(baseDir, importPath string, excludes []string) ([]Candidate, error) {
	// One invariant for the whole pipeline: baseDir is absolute past this
	// point. Anchoring, module context, and the confinement boundary must all
	// derive from the same directory, never from the process working
	// directory.
	baseDir = pathToAbs(baseDir)
	t, err := r.address(baseDir, importPath)
	if err != nil {
		return nil, err
	}
	masks, err := t.excludeMasks(excludes)
	if err != nil {
		return nil, err
	}

	var files []string
	if isGlobPattern(t.pattern) {
		files, err = globMatches(t.anchorDir, t.pattern)
		if err != nil {
			return nil, err
		}
	} else {
		full := filepath.Join(t.anchorDir, t.pattern)
		if !fileExists(full) {
			return nil, t.notFoundError(full)
		}
		files = []string{pathToAbs(full)}
	}

	kept := make([]string, 0, len(files))
	for _, file := range files {
		excluded, err := t.excludedBy(masks, file)
		if err != nil {
			return nil, err
		}
		if !excluded {
			kept = append(kept, file)
		}
	}

	candidates := make([]Candidate, 0, len(kept))
	for _, file := range kept {
		candidates = append(candidates, Candidate{
			Path:     file,
			Boundary: t.boundary,
		})
	}
	return candidates, nil
}

// excludeMasks converts exclusion patterns into slash masks relative to the
// import's anchor. An exclusion is addressed exactly like its import: a local
// import takes local patterns, a module import takes patterns inside the same
// module. Masks are pure filters over the files the import found — they never
// touch the filesystem, so a mask that matches nothing is a no-op.
func (t target) excludeMasks(excludes []string) ([]string, error) {
	masks := make([]string, 0, len(excludes))
	for _, exclude := range excludes {
		k, err := classify(exclude)
		if err != nil {
			return nil, fmt.Errorf("exclusion %q: %w", exclude, err)
		}
		if k != t.kind {
			return nil, fmt.Errorf("exclusion %q does not match the addressing of the import: exclude a module import with a module pattern and a local import with a local pattern", exclude)
		}
		mask := exclude
		if t.kind == kindModule {
			remainder, ok := strings.CutPrefix(exclude, t.modulePath+"/")
			if !ok {
				return nil, fmt.Errorf("exclusion %q must name a path inside module %s of the import", exclude, t.modulePath)
			}
			mask = remainder
		}
		mask = path.Clean(filepath.ToSlash(mask))
		if !doublestar.ValidatePattern(mask) {
			return nil, fmt.Errorf("invalid exclusion pattern %q", exclude)
		}
		masks = append(masks, mask)
	}
	return masks, nil
}

// excludedBy reports whether file — or any directory on its path relative to
// the import's anchor — matches one of the exclusion masks; a mask matching a
// directory therefore excludes its whole subtree.
func (t target) excludedBy(masks []string, file string) (bool, error) {
	if len(masks) == 0 {
		return false, nil
	}
	rel, err := filepath.Rel(t.anchorDir, file)
	if err != nil {
		return false, err
	}
	prefix := ""
	for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
		prefix = path.Join(prefix, segment)
		for _, mask := range masks {
			matched, err := doublestar.Match(mask, prefix)
			if err != nil {
				return false, fmt.Errorf("exclusion %q: %w", mask, err)
			}
			if matched {
				return true, nil
			}
		}
	}
	return false, nil
}

func (t target) notFoundError(full string) error {
	if t.kind == kindModule {
		return fmt.Errorf("module %s does not contain %s", t.modulePath, t.pattern)
	}
	return fmt.Errorf("not found at %s", full)
}

func isGlobPattern(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}
