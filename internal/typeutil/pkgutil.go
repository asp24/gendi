package typeutil

import (
	"fmt"
	"strings"
)

// SplitQualifiedName splits a qualified symbol like "pkg/path.Symbol" into package path and symbol name.
// Returns (packagePath, symbolName, error).
func SplitQualifiedName(s string) (string, string, error) {
	idx := strings.LastIndex(s, ".")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", fmt.Errorf("invalid qualified name %q", s)
	}
	return s[:idx], s[idx+1:], nil
}
