package generator

import (
	"sort"
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/internal/typeutil"
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
			pkg, _, typeParams, err := typeutil.SplitQualifiedNameWithTypeParams(svc.Constructor.Func)
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
	}
	for _, param := range cfg.Parameters {
		if param.Type != "" {
			pkgs := collectTypePackages(param.Type)
			addAll(pkgs)
		}
	}
	for _, tag := range cfg.Tags {
		if tag.ElementType != "" {
			pkgs := collectTypePackages(tag.ElementType)
			addAll(pkgs)
		}
	}

	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
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
	pkg, _, typeArgs, err := typeutil.SplitQualifiedNameWithTypeParams(typeStr)
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
