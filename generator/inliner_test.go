package generator

import (
	"go/token"
	"go/types"
	"testing"
)

type stubInliner struct {
	ok    bool
	value string
}

func (s stubInliner) TryInline(constructorDef, []string) (bool, string) {
	return s.ok, s.value
}

func newFunc(pkgPath, pkgName, name string) *types.Func {
	pkg := types.NewPackage(pkgPath, pkgName)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	return types.NewFunc(token.NoPos, pkg, name, sig)
}

func TestInlinerMakeSlice_Inlines(t *testing.T) {
	imports := NewImportManager("example.com/output")
	inliner := NewInlinerMakeSlice(imports)
	cons := constructorDef{
		kind:    "func",
		funcObj: newFunc("github.com/asp24/gendi/stdlib", "stdlib", "MakeSlice"),
		result:  types.NewSlice(types.Typ[types.Int]),
	}

	ok, call := inliner.TryInline(cons, []string{"a", "b"})
	if !ok {
		t.Fatalf("expected inliner to match MakeSlice")
	}
	if call != "[]int{a, b}" {
		t.Fatalf("unexpected inlined call: %s", call)
	}
}

func TestInlinerMakeSlice_NonStdlib(t *testing.T) {
	imports := NewImportManager("example.com/output")
	inliner := NewInlinerMakeSlice(imports)
	cons := constructorDef{
		kind:    "func",
		funcObj: newFunc("example.com/stdlib", "stdlib", "MakeSlice"),
		result:  types.NewSlice(types.Typ[types.Int]),
	}

	ok, _ := inliner.TryInline(cons, []string{"a"})
	if ok {
		t.Fatalf("expected inliner to skip non-stdlib MakeSlice")
	}
}

func TestInlinerComposite_Order(t *testing.T) {
	comp := NewInlinerComposite(
		stubInliner{ok: false, value: "nope"},
		stubInliner{ok: true, value: "first"},
		stubInliner{ok: true, value: "second"},
	)

	ok, call := comp.TryInline(constructorDef{}, []string{"a"})
	if !ok {
		t.Fatalf("expected composite to return a match")
	}
	if call != "first" {
		t.Fatalf("unexpected composite result: %s", call)
	}
}

func TestInlinerMakeChan_Inlines(t *testing.T) {
	imports := NewImportManager("example.com/output")
	inliner := NewInlinerMakeChan(imports)
	cons := constructorDef{
		kind:    "func",
		funcObj: newFunc("github.com/asp24/gendi/stdlib", "stdlib", "NewChan"),
		result:  types.NewChan(types.SendRecv, types.Typ[types.Int]),
	}

	ok, call := inliner.TryInline(cons, []string{"10"})
	if !ok {
		t.Fatalf("expected inliner to match NewChan")
	}
	if call != "make(chan int, 10)" {
		t.Fatalf("unexpected inlined call: %s", call)
	}
}

func TestInlinerMakeChan_WrongArgCount(t *testing.T) {
	imports := NewImportManager("example.com/output")
	inliner := NewInlinerMakeChan(imports)
	cons := constructorDef{
		kind:    "func",
		funcObj: newFunc("github.com/asp24/gendi/stdlib", "stdlib", "NewChan"),
		result:  types.NewChan(types.SendRecv, types.Typ[types.Int]),
	}

	ok, _ := inliner.TryInline(cons, nil)
	if ok {
		t.Fatalf("expected inliner to skip when size arg is missing")
	}
}
