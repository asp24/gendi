package yaml

import (
	"fmt"
	"strings"

	di "github.com/gendi-org/gendi"
)

// ParseArgumentString analyzes a string to determine if it represents a special argument kind
// (like a service reference, parameter, or tagged collection) or if it is a literal string.
// Strings that start with a reserved "!" sigil but do not form a valid argument are rejected
// instead of silently becoming literals.
func ParseArgumentString(s string) (di.ArgumentKind, string, error) {
	switch {
	case s == "@.inner":
		return di.ArgInner, s, nil
	case len(s) > 1 && s[0] == '@':
		return di.ArgServiceRef, s[1:], nil
	case len(s) > 2 && s[0] == '%' && s[len(s)-1] == '%':
		return di.ArgParam, s[1 : len(s)-1], nil
	case len(s) > len("!spread:") && s[:len("!spread:")] == "!spread:":
		return di.ArgSpread, s[len("!spread:"):], nil
	case len(s) > len("!tagged:") && s[:len("!tagged:")] == "!tagged:":
		return di.ArgTagged, s[len("!tagged:"):], nil
	case len(s) > len("!go:") && s[:len("!go:")] == "!go:":
		return di.ArgGoRef, s[len("!go:"):], nil
	case len(s) > len("!field:@") && s[:len("!field:@")] == "!field:@":
		return di.ArgFieldAccessService, s[len("!field:@"):], nil
	case len(s) > len("!field:!go:") && s[:len("!field:!go:")] == "!field:!go:":
		return di.ArgFieldAccessGo, s[len("!field:!go:"):], nil
	case strings.HasPrefix(s, "!field:"):
		return di.ArgLiteral, "", fmt.Errorf("invalid argument %q: !field: must be followed by @service.Field or !go:pkg.Symbol.Field", s)
	case strings.HasPrefix(s, "!spread:"), strings.HasPrefix(s, "!tagged:"), strings.HasPrefix(s, "!go:"):
		return di.ArgLiteral, "", fmt.Errorf("invalid argument %q: missing value after the %q prefix", s, s)
	}
	return di.ArgLiteral, "", nil
}

// IsServiceAlias checks if a string indicates a service alias (starts with @).
func IsServiceAlias(ref string) bool {
	return len(ref) > 1 && ref[0] == '@'
}

// ParseServiceAlias extracts the service ID from an alias string (removes @ prefix).
func ParseServiceAlias(ref string) string {
	return ref[1:]
}
