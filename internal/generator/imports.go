package generator

import (
	"fmt"
	"sort"
	"strings"

	"go/types"
)

type importManager struct {
	aliases      map[string]string
	used         map[string]bool
	outputPkgPath string
}

func newImportManager(outputPkgPath string) *importManager {
	return &importManager{
		aliases:      map[string]string{},
		used:         map[string]bool{},
		outputPkgPath: outputPkgPath,
	}
}

func (m *importManager) qualifier(pkg *types.Package) string {
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

func (m *importManager) aliasInUse(alias string) bool {
	for _, v := range m.aliases {
		if v == alias {
			return true
		}
	}
	return false
}

func (m *importManager) typeString(t types.Type) string {
	return types.TypeString(t, m.qualifier)
}

func (m *importManager) funcName(fn *types.Func) string {
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

func (m *importManager) renderImports(extra []string) string {
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

