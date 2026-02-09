package generator

import (
	"go/types"
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/typeres"
	"github.com/asp24/gendi/xmaps"
)

func collectPackagePaths(cfg *di.Config) ([]string, error) {
	seen := map[string]bool{}
	add := func(path string) {
		if path != "" {
			seen[path] = true
		}
	}
	addAll := func(paths []string) {
		for _, p := range paths {
			add(p)
		}
	}

	for _, svc := range cfg.Services {
		if svc.Constructor.Func != "" {
			// Extract function package and type arguments
			pkg, _, typeParams, err := typeres.SplitQualifiedNameWithTypeParams(svc.Constructor.Func)
			if err != nil {
				return nil, err
			}
			add(pkg)

			// Collect packages from type arguments
			for _, tp := range typeParams {
				pkgs := collectTypePackages(tp)
				addAll(pkgs)
			}
		}
		if svc.Type != "" {
			pkgs := collectTypePackages(svc.Type)
			addAll(pkgs)
		}
		// Collect packages from !go: argument references
		for _, arg := range svc.Constructor.Args {
			if arg.Kind == di.ArgGoRef {
				pkg, _, _, err := typeres.SplitQualifiedNameWithTypeParams(arg.Value)
				if err == nil {
					add(pkg)
				}
			}
		}
	}
	for _, param := range cfg.Parameters {
		if param.Type != "" {
			pkgs := collectTypePackages(param.Type)
			addAll(pkgs)
		}
	}

	// Check if there are tags or tagged arguments - need stdlib for MakeSlice
	hasTagsOrTaggedArgs := len(cfg.Tags) > 0
	if !hasTagsOrTaggedArgs {
		for _, svc := range cfg.Services {
			if len(svc.Tags) > 0 {
				hasTagsOrTaggedArgs = true
				break
			}
			for _, arg := range svc.Constructor.Args {
				if arg.Kind == di.ArgTagged {
					hasTagsOrTaggedArgs = true
					break
				}
			}
		}
	}
	if hasTagsOrTaggedArgs {
		// Tag desugaring uses stdlib.MakeSlice
		add("github.com/asp24/gendi/stdlib")
	}

	for _, tag := range cfg.Tags {
		if tag.ElementType != "" {
			pkgs := collectTypePackages(tag.ElementType)
			addAll(pkgs)
		}
	}

	return xmaps.OrderedKeys(seen), nil
}

// collectTypePackages extracts all package paths from a type string,
// including composite types like chan, slice, map, pointers, and arrays.
func collectTypePackages(typeStr string) []string {
	typeStr = strings.TrimSpace(typeStr)
	var result []string

	// Pointer type: *T
	if strings.HasPrefix(typeStr, "*") {
		return collectTypePackages(typeStr[1:])
	}

	// Slice type: []T
	if strings.HasPrefix(typeStr, "[]") {
		return collectTypePackages(typeStr[2:])
	}

	// Array type: [N]T
	if strings.HasPrefix(typeStr, "[") {
		closeBracket := strings.Index(typeStr, "]")
		if closeBracket != -1 {
			return collectTypePackages(typeStr[closeBracket+1:])
		}
		return nil
	}

	// Map type: map[K]V
	if strings.HasPrefix(typeStr, "map[") {
		keyEnd := findMatchingBracketHelper(typeStr, 3)
		if keyEnd != -1 {
			keyStr := typeStr[4:keyEnd]
			valStr := typeStr[keyEnd+1:]
			result = append(result, collectTypePackages(keyStr)...)
			result = append(result, collectTypePackages(valStr)...)
			return result
		}
		return nil
	}

	// Channel types: <-chan T, chan<- T, chan T
	if strings.HasPrefix(typeStr, "<-chan ") {
		return collectTypePackages(typeStr[7:])
	}
	if strings.HasPrefix(typeStr, "chan<- ") {
		return collectTypePackages(typeStr[7:])
	}
	if strings.HasPrefix(typeStr, "chan ") {
		return collectTypePackages(typeStr[5:])
	}

	// Basic types have no package
	if !strings.Contains(typeStr, ".") {
		return nil
	}

	// Named type: pkg/path.TypeName or pkg/path.TypeName[T1, T2]
	pkg, _, typeArgs, err := typeres.SplitQualifiedNameWithTypeParams(typeStr)
	if err != nil {
		return nil
	}

	result = append(result, pkg)

	// Recursively collect packages from type arguments
	for _, arg := range typeArgs {
		result = append(result, collectTypePackages(arg)...)
	}

	return result
}

// findMatchingBracketHelper is a helper for collectTypePackages
func findMatchingBracketHelper(s string, start int) int {
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

func isNilable(t types.Type) bool {
	switch tt := t.(type) {
	case *types.Pointer, *types.Interface, *types.Slice, *types.Map, *types.Chan, *types.Signature:
		return true
	case *types.Named:
		return isNilable(tt.Underlying())
	case *types.Alias:
		return isNilable(tt.Underlying())
	default:
		return false
	}
}
