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

func NewImportManager(outputPkgPath string) *ImportManager {
	return &ImportManager{
		aliases:       map[string]string{},
		used:          map[string]bool{},
		outputPkgPath: outputPkgPath,
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
	for _, v := range m.aliases {
		if v == alias {
			return true
		}
	}
	return false
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
