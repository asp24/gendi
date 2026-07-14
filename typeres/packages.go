package typeres

import "strings"

// CollectTypePackages extracts all package paths from a type string,
// including composite types like chan, slice, map, pointers, and arrays.
func CollectTypePackages(typeStr string) []string {
	typeStr = strings.TrimSpace(typeStr)

	// Pointer type: *T
	if strings.HasPrefix(typeStr, "*") {
		return CollectTypePackages(typeStr[1:])
	}

	// Slice type: []T
	if strings.HasPrefix(typeStr, "[]") {
		return CollectTypePackages(typeStr[2:])
	}

	// Array type: [N]T
	if strings.HasPrefix(typeStr, "[") {
		closeBracket := strings.Index(typeStr, "]")
		if closeBracket != -1 {
			return CollectTypePackages(typeStr[closeBracket+1:])
		}
		return nil
	}

	// Map type: map[K]V
	if strings.HasPrefix(typeStr, "map[") {
		keyEnd := findMatchingBracket(typeStr, 3)
		if keyEnd != -1 {
			keyStr := typeStr[4:keyEnd]
			valStr := typeStr[keyEnd+1:]
			var result []string
			result = append(result, CollectTypePackages(keyStr)...)
			result = append(result, CollectTypePackages(valStr)...)
			return result
		}
		return nil
	}

	// Channel types: <-chan T, chan<- T, chan T
	if strings.HasPrefix(typeStr, "<-chan ") {
		return CollectTypePackages(typeStr[7:])
	}
	if strings.HasPrefix(typeStr, "chan<- ") {
		return CollectTypePackages(typeStr[7:])
	}
	if strings.HasPrefix(typeStr, "chan ") {
		return CollectTypePackages(typeStr[5:])
	}

	// Basic types have no package
	if !strings.Contains(typeStr, ".") {
		return nil
	}

	// Named type: pkg/path.TypeName or pkg/path.TypeName[T1, T2]
	pkg, _, typeArgs, err := SplitQualifiedNameWithTypeParams(typeStr)
	if err != nil {
		return nil
	}

	var result []string
	result = append(result, pkg)

	// Recursively collect packages from type arguments
	for _, arg := range typeArgs {
		result = append(result, CollectTypePackages(arg)...)
	}

	return result
}

// CollectFuncPackages extracts the package path and type argument packages
// from a qualified function name like "pkg/path.Func[T1, T2]".
func CollectFuncPackages(funcStr string) []string {
	if funcStr == "" {
		return nil
	}
	pkg, _, typeParams, err := SplitQualifiedNameWithTypeParams(funcStr)
	if err != nil {
		return nil
	}
	var result []string
	if pkg != "" {
		result = append(result, pkg)
	}
	for _, tp := range typeParams {
		result = append(result, CollectTypePackages(tp)...)
	}
	return result
}

// CollectGoRefPackages extracts the package path from a Go reference value
// like "pkg/path.Symbol" used in !go: arguments.
func CollectGoRefPackages(value string) []string {
	pkg, _, _, err := SplitQualifiedNameWithTypeParams(value)
	if err != nil || pkg == "" {
		return nil
	}
	return []string{pkg}
}

// CollectFieldAccessGoPackages extracts candidate package paths from a field
// access on a Go symbol like "pkg/path.Symbol.Field.SubField". The boundary
// between the package path, the symbol, and the field chain cannot be
// determined statically (package paths may contain dots, e.g.
// "gopkg.in/yaml.v3"), so every plausible prefix is returned. Candidates must
// be loaded leniently: only one of them is a real package.
func CollectFieldAccessGoPackages(value string) []string {
	parts := strings.Split(value, ".")
	var result []string
	seen := map[string]bool{}
	for i := len(parts) - 1; i >= 2; i-- {
		qualName := strings.Join(parts[:i], ".")
		pkg, _, _, err := SplitQualifiedNameWithTypeParams(qualName)
		if err == nil && pkg != "" && !seen[pkg] {
			seen[pkg] = true
			result = append(result, pkg)
		}
	}
	return result
}
