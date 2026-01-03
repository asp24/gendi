package typeres

import (
	"fmt"
	"strings"
)

// SplitQualifiedNameWithTypeParams splits a qualified symbol with optional type parameters.
// Input: "pkg/path.Symbol[type1, type2]"
// Returns: (packagePath, symbolName, typeParams, error)
func SplitQualifiedNameWithTypeParams(s string) (string, string, []string, error) {
	var typeParams []string

	// Find type parameters
	bracketIdx := strings.Index(s, "[")
	if bracketIdx != -1 {
		if !strings.HasSuffix(s, "]") {
			return "", "", nil, fmt.Errorf("invalid type parameters in %q: missing closing bracket", s)
		}
		typeParamsStr := s[bracketIdx+1 : len(s)-1]
		typeParams = splitTypeParams(typeParamsStr)
		s = s[:bracketIdx]
	}

	idx := strings.LastIndex(s, ".")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", nil, fmt.Errorf("invalid qualified name %q", s)
	}

	return s[:idx], s[idx+1:], typeParams, nil
}

// splitTypeParams splits comma-separated type parameters, respecting nested brackets.
// "Type1, Map[string, int], Type2" -> ["Type1", "Map[string, int]", "Type2"]
func splitTypeParams(s string) []string {
	var params []string
	var current strings.Builder
	depth := 0

	for _, r := range s {
		switch r {
		case '[':
			depth++
			current.WriteRune(r)
		case ']':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				params = append(params, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		params = append(params, strings.TrimSpace(current.String()))
	}

	return params
}
