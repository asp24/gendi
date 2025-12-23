package generator

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	di "github.com/asp24/gendi"
)

func splitPkgSymbol(s string) (string, string, error) {
	idx := strings.LastIndex(s, ".")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", fmt.Errorf("invalid qualified name %q", s)
	}
	return s[:idx], s[idx+1:], nil
}

func paramGetterMethod(t types.Type) (string, error) {
	switch {
	case types.Identical(t, types.Typ[types.String]):
		return "GetString", nil
	case types.Identical(t, types.Typ[types.Int]):
		return "GetInt", nil
	case types.Identical(t, types.Typ[types.Bool]):
		return "GetBool", nil
	case types.Identical(t, types.Typ[types.Float64]):
		return "GetFloat", nil
	case isTimeDuration(t):
		return "GetDuration", nil
	default:
		return "", fmt.Errorf("unsupported parameter type %s", types.TypeString(t, nil))
	}
}

func collectPackagePaths(cfg *di.Config) ([]string, error) {
	seen := map[string]bool{}
	add := func(path string) {
		if path != "" {
			seen[path] = true
		}
	}

	for _, svc := range cfg.Services {
		if svc.Constructor.Func != "" {
			pkg, _, err := splitPkgSymbol(svc.Constructor.Func)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
		if svc.Type != "" {
			pkg, err := typePkgPath(svc.Type)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
	}
	for _, param := range cfg.Parameters {
		if param.Type != "" {
			pkg, err := typePkgPath(param.Type)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
	}
	for _, tag := range cfg.Tags {
		if tag.ElementType != "" {
			pkg, err := typePkgPath(tag.ElementType)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
	}

	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func typePkgPath(typeStr string) (string, error) {
	t := strings.TrimPrefix(typeStr, "*")
	if !strings.Contains(t, ".") {
		return "", nil
	}
	pkg, _, err := splitPkgSymbol(t)
	if err != nil {
		return "", err
	}
	return pkg, nil
}
