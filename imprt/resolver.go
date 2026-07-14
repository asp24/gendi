package imprt

// Resolver resolves import paths to file paths.
type Resolver interface {
	// CanResolve is a cheap syntactic pre-filter: it reports whether the
	// import path has the shape this resolver handles (glob, absolute, ...).
	// It must not touch the filesystem.
	CanResolve(importPath string) bool
	// Resolve resolves the import path to a list of file paths. Returning
	// (nil, nil) means "not mine, let the next resolver in the chain try";
	// see ResolverComposite.Resolve.
	Resolve(baseDir, importPath string) ([]string, error)
}
