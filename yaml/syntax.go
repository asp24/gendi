package yaml

import (
	di "github.com/asp24/gendi"
)

// ParseArgumentString analyzes a string to determine if it represents a special argument kind
// (like a service reference, parameter, or tagged collection) or if it is a literal string.
func ParseArgumentString(s string) (di.ArgumentKind, string) {
	switch {
	case s == "@.inner":
		return di.ArgInner, s
	case len(s) > 1 && s[0] == '@':
		return di.ArgServiceRef, s[1:]
	case len(s) > 2 && s[0] == '%' && s[len(s)-1] == '%':
		return di.ArgParam, s[1 : len(s)-1]
	case len(s) > len("!spread:") && s[:len("!spread:")] == "!spread:":
		return di.ArgSpread, s[len("!spread:"):]
	case len(s) > len("!tagged:") && s[:len("!tagged:")] == "!tagged:":
		return di.ArgTagged, s[len("!tagged:"):]
	case len(s) > len("!go:") && s[:len("!go:")] == "!go:":
		return di.ArgGoRef, s[len("!go:"):]
	case len(s) > len("!field:") && s[:len("!field:")] == "!field:":
		return di.ArgFieldAccess, s[len("!field:"):]
	}
	return di.ArgLiteral, ""
}

// IsServiceAlias checks if a string indicates a service alias (starts with @).
func IsServiceAlias(ref string) bool {
	return len(ref) > 1 && ref[0] == '@'
}

// ParseServiceAlias extracts the service ID from an alias string (removes @ prefix).
func ParseServiceAlias(ref string) string {
	return ref[1:]
}