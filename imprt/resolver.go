package imprt

// Resolution is the result of resolving an import path.
type Resolution struct {
	// Files are the absolute paths of the resolved config files.
	Files []string
	// BaseDir is the directory the resolved files were located from: the
	// module root for module imports, the glob base for absolute globs, or
	// the importing file's directory for local imports. Relative exclusion
	// patterns are resolved against it.
	BaseDir string
}

// Resolver resolves import paths to file paths.
type Resolver interface {
	// CanResolve is a cheap syntactic pre-filter: it reports whether the
	// import path has the shape this resolver handles (glob, absolute, ...).
	// It must not touch the filesystem.
	CanResolve(importPath string) bool
	// Resolve resolves the import path to the matched file paths plus the
	// base directory they were resolved under. Returning (nil, nil) means
	// "not mine, let the next resolver in the chain try"; see
	// ResolverComposite.Resolve.
	Resolve(baseDir, importPath string) (*Resolution, error)
}
