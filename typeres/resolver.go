package typeres

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"
)

// Resolver coordinates package loading and type resolution.
// It implements the ir.TypeResolver interface.
type Resolver struct {
	cache         *Cache
	outputPkgPath string
}

// NewResolver creates a new Resolver with the given module root and output package path.
func NewResolver(moduleRoot, outputPkgPath string) *Resolver {
	return &Resolver{
		cache:         NewCache(moduleRoot),
		outputPkgPath: outputPkgPath,
	}
}

// LookupFunc looks up a function by package path and name.
func (r *Resolver) LookupFunc(pkgPath, name string) (*types.Func, error) {
	pkg, err := r.cache.Get(pkgPath)
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
// Supports composite types: pointers (*T), slices ([]T), arrays ([N]T),
// maps (map[K]V), channels (chan T, <-chan T, chan<- T), and named types.
func (r *Resolver) LookupType(typeStr string) (types.Type, error) {
	typeStr = strings.TrimSpace(typeStr)

	// Pointer type: *T
	if strings.HasPrefix(typeStr, "*") {
		elem, err := r.LookupType(typeStr[1:])
		if err != nil {
			return nil, err
		}
		return types.NewPointer(elem), nil
	}

	// Slice type: []T
	if strings.HasPrefix(typeStr, "[]") {
		elem, err := r.LookupType(typeStr[2:])
		if err != nil {
			return nil, err
		}
		return types.NewSlice(elem), nil
	}

	// Array type: [N]T
	if strings.HasPrefix(typeStr, "[") {
		closeBracket := strings.Index(typeStr, "]")
		if closeBracket == -1 {
			return nil, fmt.Errorf("invalid array type %q: missing ]", typeStr)
		}
		sizeStr := typeStr[1:closeBracket]
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid array size in %q: %w", typeStr, err)
		}
		elem, err := r.LookupType(typeStr[closeBracket+1:])
		if err != nil {
			return nil, err
		}
		return types.NewArray(elem, size), nil
	}

	// Map type: map[K]V
	if strings.HasPrefix(typeStr, "map[") {
		keyEnd := findMatchingBracket(typeStr, 3) // Start after "map["
		if keyEnd == -1 {
			return nil, fmt.Errorf("invalid map type %q: missing ]", typeStr)
		}
		keyStr := typeStr[4:keyEnd]
		valStr := typeStr[keyEnd+1:]

		key, err := r.LookupType(keyStr)
		if err != nil {
			return nil, fmt.Errorf("map key type: %w", err)
		}
		val, err := r.LookupType(valStr)
		if err != nil {
			return nil, fmt.Errorf("map value type: %w", err)
		}
		return types.NewMap(key, val), nil
	}

	// Receive-only channel: <-chan T
	if strings.HasPrefix(typeStr, "<-chan ") {
		elem, err := r.LookupType(typeStr[7:])
		if err != nil {
			return nil, err
		}
		return types.NewChan(types.RecvOnly, elem), nil
	}

	// Send-only channel: chan<- T
	if strings.HasPrefix(typeStr, "chan<- ") {
		elem, err := r.LookupType(typeStr[7:])
		if err != nil {
			return nil, err
		}
		return types.NewChan(types.SendOnly, elem), nil
	}

	// Bidirectional channel: chan T
	if strings.HasPrefix(typeStr, "chan ") {
		elem, err := r.LookupType(typeStr[5:])
		if err != nil {
			return nil, err
		}
		return types.NewChan(types.SendRecv, elem), nil
	}

	// Basic/universe types (int, string, bool, etc.)
	if basic := types.Universe.Lookup(typeStr); basic != nil {
		return basic.Type(), nil
	}

	// Named type: pkg/path.TypeName or pkg/path.TypeName[T1, T2]
	pkgPath, name, typeArgStrs, err := SplitQualifiedNameWithTypeParams(typeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid type %q: %w", typeStr, err)
	}

	pkg, err := r.cache.Get(pkgPath)
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

	baseType := typeObj.Type()

	// Check if the type is generic (has type parameters)
	named, isNamed := baseType.(*types.Named)
	if isNamed {
		typeParams := named.TypeParams()
		if typeParams != nil && typeParams.Len() > 0 {
			// Type is generic - must have type arguments
			if len(typeArgStrs) == 0 {
				return nil, fmt.Errorf("generic type %s.%s requires %d type argument(s)",
					pkgPath, name, typeParams.Len())
			}
			if len(typeArgStrs) != typeParams.Len() {
				return nil, fmt.Errorf("generic type %s.%s expects %d type arguments, got %d",
					pkgPath, name, typeParams.Len(), len(typeArgStrs))
			}

			// Resolve type arguments
			typeArgs := make([]types.Type, len(typeArgStrs))
			for i, argStr := range typeArgStrs {
				t, err := r.LookupType(argStr)
				if err != nil {
					return nil, fmt.Errorf("type argument %d: %w", i, err)
				}
				typeArgs[i] = t
			}

			// Instantiate the generic type
			instantiated, err := types.Instantiate(nil, named, typeArgs, true)
			if err != nil {
				return nil, fmt.Errorf("instantiate %s.%s: %w", pkgPath, name, err)
			}
			return instantiated, nil
		}
	}

	// Non-generic type should not have type arguments
	if len(typeArgStrs) > 0 {
		return nil, fmt.Errorf("type %s.%s is not generic but has type arguments", pkgPath, name)
	}

	return baseType, nil
}

// findMatchingBracket finds the index of the closing bracket that matches
// the opening bracket at position start. Returns -1 if not found.
func findMatchingBracket(s string, start int) int {
	depth := 1
	for i := start + 1; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// LookupMethod looks up a method on a type.
func (r *Resolver) LookupMethod(recv types.Type, name string) (*types.Func, error) {
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

// InstantiateFunc instantiates a generic function with the given type arguments.
// Returns the instantiated signature and resolved type arguments.
func (r *Resolver) InstantiateFunc(fn *types.Func, typeArgStrs []string) (*types.Signature, []types.Type, error) {
	sig := fn.Type().(*types.Signature)

	// Get type parameters from the signature
	typeParams := sig.TypeParams()
	if typeParams == nil || typeParams.Len() == 0 {
		return nil, nil, fmt.Errorf("function %s is not generic", fn.Name())
	}

	if len(typeArgStrs) != typeParams.Len() {
		return nil, nil, fmt.Errorf("function %s expects %d type arguments, got %d",
			fn.Name(), typeParams.Len(), len(typeArgStrs))
	}

	// Resolve type argument strings to types
	typeArgs := make([]types.Type, len(typeArgStrs))
	for i, typeStr := range typeArgStrs {
		t, err := r.LookupType(typeStr)
		if err != nil {
			return nil, nil, fmt.Errorf("type argument %d: %w", i, err)
		}
		typeArgs[i] = t
	}

	// Instantiate the generic signature
	instantiated, err := types.Instantiate(nil, sig, typeArgs, true)
	if err != nil {
		return nil, nil, fmt.Errorf("instantiate %s: %w", fn.Name(), err)
	}

	instSig, ok := instantiated.(*types.Signature)
	if !ok {
		return nil, nil, fmt.Errorf("instantiated type is not a signature")
	}

	return instSig, typeArgs, nil
}

// LoadPackages loads the specified packages into the cache.
func (r *Resolver) LoadPackages(paths []string) error {
	return r.cache.Load(paths)
}

// OutputPkgPath returns the output package path.
func (r *Resolver) OutputPkgPath() string {
	return r.outputPkgPath
}
