package generator

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/asp24/gendi/internal/typeutil"
)

// TypeLoader coordinates package loading and type resolution.
// It implements the ir.TypeResolver interface.
type TypeLoader struct {
	cache         *PackageCache
	outputPkgPath string
}

// NewTypeLoader creates a new TypeLoader with the given options.
// Options must be finalized before calling this (via Options.Finalize()).
func NewTypeLoader(opts Options) (*TypeLoader, error) {
	return &TypeLoader{
		cache:         NewPackageCache(opts.ModuleRoot),
		outputPkgPath: opts.OutputPkgPath,
	}, nil
}

// LookupFunc looks up a function by package path and name.
func (l *TypeLoader) LookupFunc(pkgPath, name string) (*types.Func, error) {
	pkg, err := l.cache.Get(pkgPath)
	if err != nil {
		return nil, err
	}

	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("symbol %s not found in %s", name, pkgPath)
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%s is not a function in %s", name, pkgPath)
	}

	return fn, nil
}

// LookupType resolves a type string to a types.Type.
func (l *TypeLoader) LookupType(typeStr string) (types.Type, error) {
	ptr := false
	if strings.HasPrefix(typeStr, "*") {
		ptr = true
		typeStr = strings.TrimPrefix(typeStr, "*")
	}

	if basic := types.Universe.Lookup(typeStr); basic != nil {
		t := basic.Type()
		if ptr {
			return types.NewPointer(t), nil
		}
		return t, nil
	}

	pkgPath, name, err := typeutil.SplitQualifiedName(typeStr)
	if err != nil {
		return nil, err
	}

	pkg, err := l.cache.Get(pkgPath)
	if err != nil {
		return nil, err
	}

	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("type %s not found in %s", name, pkgPath)
	}

	typeObj, ok := obj.(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("%s is not a type in %s", name, pkgPath)
	}

	t := typeObj.Type()
	if ptr {
		return types.NewPointer(t), nil
	}

	return t, nil
}

// LookupMethod looks up a method on a type.
func (l *TypeLoader) LookupMethod(recv types.Type, name string) (*types.Func, error) {
	obj, _, _ := types.LookupFieldOrMethod(recv, true, nil, name)
	if obj == nil {
		return nil, fmt.Errorf("method %s not found", name)
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%s is not a method", name)
	}

	return fn, nil
}

// typeString formats a types.Type as a string with appropriate package qualification.
func (l *TypeLoader) typeString(t types.Type) string {
	return FormatTypeString(t, l.outputPkgPath)
}

// loadPackages loads the specified packages into the cache.
func (l *TypeLoader) loadPackages(paths []string) error {
	return l.cache.Load(paths)
}
