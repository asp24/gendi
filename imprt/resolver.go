package imprt

// Resolver resolves import paths to file paths.
type Resolver interface {
	// CanResolve returns true if this resolver can handle the given import path.
	CanResolve(importPath string) bool
	// Resolve resolves the import path to a list of file paths.
	Resolve(baseDir, importPath string) ([]string, error)
}
