package generator

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
)

type typeLoader struct {
	fset          *token.FileSet
	packages      map[string]*types.Package
	outputPkgPath string
}

func newTypeLoader(opts Options) (*typeLoader, error) {
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

	return &typeLoader{
		fset:          token.NewFileSet(),
		packages:      map[string]*types.Package{},
		outputPkgPath: outputPath,
	}, nil
}

func (l *typeLoader) ensurePackage(path string) (*types.Package, error) {
	if pkg, ok := l.packages[path]; ok {
		return pkg, nil
	}

	buildPkg, err := build.Default.Import(path, "", 0)
	if err != nil {
		return nil, fmt.Errorf("import %q: %w", path, err)
	}

	files := make([]*ast.File, 0, len(buildPkg.GoFiles))
	for _, name := range buildPkg.GoFiles {
		filePath := filepath.Join(buildPkg.Dir, name)
		file, err := parser.ParseFile(l.fset, filePath, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filePath, err)
		}
		files = append(files, file)
	}

	conf := types.Config{Importer: importer.Default(), Error: func(err error) {}}
	info := &types.Info{
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
		Types: map[ast.Expr]types.TypeAndValue{},
	}
	pkg, err := conf.Check(path, l.fset, files, info)
	if err != nil {
		return nil, fmt.Errorf("type-check %q: %w", path, err)
	}
	l.packages[path] = pkg
	return pkg, nil
}

func (l *typeLoader) lookupFunc(pkgPath, name string) (*types.Func, error) {
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

func (l *typeLoader) lookupType(typeStr string) (types.Type, error) {
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

func (l *typeLoader) lookupMethod(recv types.Type, name string) (*types.Func, error) {
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

func (l *typeLoader) typeString(t types.Type) string {
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

