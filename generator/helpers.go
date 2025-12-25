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

	for _, svc := range cfg.Services {
		if svc.Constructor.Func != "" {
			pkg, _, err := typeutil.SplitQualifiedName(svc.Constructor.Func)
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
	pkg, _, err := typeutil.SplitQualifiedName(t)
	if err != nil {
		return "", err
	}
	return pkg, nil
}
