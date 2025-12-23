package generator

import (
	"bufio"
	"errors"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type TypeLoader struct {
	packages      map[string]*types.Package
	outputPkgPath string
	moduleRoot    string
}

func NewTypeLoader(opts Options) (*TypeLoader, error) {
	modPath, modRoot := opts.ModulePath, opts.ModuleRoot
	if modPath == "" || modRoot == "" {
		path, root, err := moduleInfo()
		if err != nil {
			return nil, err
		}
		modPath, modRoot = path, root
	}
	outputPath := opts.OutputPkgPath
	if outputPath == "" {
		outputPath = outputPkgPath(modPath, modRoot, opts.Out)
	}

	return &TypeLoader{
		packages:      map[string]*types.Package{},
		outputPkgPath: outputPath,
		moduleRoot:    modRoot,
	}, nil
}

func (l *TypeLoader) ensurePackage(path string) (*types.Package, error) {
	if pkg, ok := l.packages[path]; ok {
		return pkg, nil
	}
	return nil, fmt.Errorf("package %q not loaded", path)
}

// LookupFunc looks up a function by package path and name.
func (l *TypeLoader) LookupFunc(pkgPath, name string) (*types.Func, error) {
	pkg, err := l.ensurePackage(pkgPath)
	if err != nil {
		return nil, err
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("symbol %s not found in %s", name, pkgPath)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%s is not a function in %s", name, pkgPath)
	}
	return fn, nil
}

// LookupType resolves a type string to a types.Type.
func (l *TypeLoader) LookupType(typeStr string) (types.Type, error) {
	ptr := false
	if strings.HasPrefix(typeStr, "*") {
		ptr = true
		typeStr = strings.TrimPrefix(typeStr, "*")
	}
	if basic := types.Universe.Lookup(typeStr); basic != nil {
		t := basic.Type()
		if ptr {
			return types.NewPointer(t), nil
		}
		return t, nil
	}
	pkgPath, name, err := splitPkgSymbol(typeStr)
	if err != nil {
		return nil, err
	}
	pkg, err := l.ensurePackage(pkgPath)
	if err != nil {
		return nil, err
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("type %s not found in %s", name, pkgPath)
	}
	typeObj, ok := obj.(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("%s is not a type in %s", name, pkgPath)
	}
	t := typeObj.Type()
	if ptr {
		return types.NewPointer(t), nil
	}
	return t, nil
}

// LookupMethod looks up a method on a type.
func (l *TypeLoader) LookupMethod(recv types.Type, name string) (*types.Func, error) {
	obj, _, _ := types.LookupFieldOrMethod(recv, true, nil, name)
	if obj == nil {
		return nil, fmt.Errorf("method %s not found", name)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%s is not a method", name)
	}
	return fn, nil
}

func (l *TypeLoader) typeString(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string {
		if pkg.Path() == l.outputPkgPath {
			return ""
		}
		return pkg.Name()
	})
}

func moduleInfo() (string, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", "", errors.New("go.mod not found")
		}
		root = parent
	}
	file, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), root, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return "", "", errors.New("module path not found in go.mod")
}

func outputPkgPath(modPath, modRoot, out string) string {
	outDir := out
	if strings.HasSuffix(outDir, ".go") {
		outDir = filepath.Dir(outDir)
	}
	outDir, _ = filepath.Abs(outDir)
	rel, err := filepath.Rel(modRoot, outDir)
	if err != nil {
		return modPath
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return modPath
	}
	return modPath + "/" + rel
}

func (l *TypeLoader) loadPackages(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports |
			packages.NeedDeps,
		Dir: l.moduleRoot,
	}
	pkgs, err := packages.Load(cfg, paths...)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}
	if err := packagesLoadError(pkgs); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, pkg := range pkgs {
		cachePackageTree(l.packages, seen, pkg)
	}
	return nil
}

func cachePackageTree(cache map[string]*types.Package, seen map[string]bool, pkg *packages.Package) {
	if pkg == nil {
		return
	}
	key := pkg.PkgPath
	if key == "" {
		key = pkg.ID
	}
	if key != "" {
		if seen[key] {
			return
		}
		seen[key] = true
	}
	if pkg.Types != nil {
		if key != "" {
			cache[key] = pkg.Types
		}
	}
	for _, imp := range pkg.Imports {
		cachePackageTree(cache, seen, imp)
	}
}

func packagesLoadError(pkgs []*packages.Package) error {
	var errs []string
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}
