package generator

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

type ImportManager struct {
	aliases       map[string]string
	used          map[string]bool
	outputPkgPath string
}

func NewImportManager(outputPkgPath string, reservedAliases ...string) *ImportManager {
	m := &ImportManager{
		aliases:       map[string]string{},
		used:          map[string]bool{},
		outputPkgPath: outputPkgPath,
	}
	m.ReserveAliases(reservedAliases...)
	return m
}

// ReserveAliases marks aliases as used so they cannot be selected.
func (m *ImportManager) ReserveAliases(aliases ...string) {
	for _, alias := range aliases {
		if alias == "" {
			continue
		}
		m.used[alias] = true
	}
}

func (m *ImportManager) qualifier(pkg *types.Package) string {
	if pkg.Path() == m.outputPkgPath {
		return ""
	}
	if alias, ok := m.aliases[pkg.Path()]; ok {
		return alias
	}
	base := pkg.Name()
	alias := base
	if alias == "" {
		alias = "pkg"
	}
	if m.aliasInUse(alias) {
		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s%d", alias, i)
			if !m.aliasInUse(candidate) {
				alias = candidate
				break
			}
		}
	}
	m.aliases[pkg.Path()] = alias
	m.used[alias] = true
	return alias
}

func (m *ImportManager) aliasInUse(alias string) bool {
	return m.used[alias]
}

func (m *ImportManager) typeString(t types.Type) string {
	return types.TypeString(t, m.qualifier)
}

func (m *ImportManager) funcName(fn *types.Func) string {
	pkg := fn.Pkg()
	if pkg == nil {
		return fn.Name()
	}
	alias := m.qualifier(pkg)
	if alias == "" {
		return fn.Name()
	}
	return alias + "." + fn.Name()
}

// funcNameWithTypeArgs returns the function name with type arguments for generic functions.
func (m *ImportManager) funcNameWithTypeArgs(fn *types.Func, typeArgs []types.Type) string {
	name := m.funcName(fn)
	if len(typeArgs) == 0 {
		return name
	}

	// Build type arguments string
	typeArgStrs := make([]string, len(typeArgs))
	for i, t := range typeArgs {
		typeArgStrs[i] = m.typeString(t)
	}

	return name + "[" + strings.Join(typeArgStrs, ", ") + "]"
}

func (m *ImportManager) renderImports(extra []string) string {
	imports := []string{}
	for path, alias := range m.aliases {
		if alias == "" {
			imports = append(imports, fmt.Sprintf("\t\"%s\"\n", path))
			continue
		}
		imports = append(imports, fmt.Sprintf("\t%s \"%s\"\n", alias, path))
	}
	for _, path := range extra {
		imports = append(imports, fmt.Sprintf("\t\"%s\"\n", path))
	}
	sort.Strings(imports)
	if len(imports) == 0 {
		return ""
	}
	return "import (\n" + strings.Join(imports, "") + ")\n\n"
}
