package typeres

import (
	"go/types"
	"testing"
)

// structFieldType looks up struct type structName in pkgPath and returns the
// type of the named field, failing the test if anything is missing.
func structFieldType(t *testing.T, c *Cache, pkgPath, structName, field string) types.Type {
	t.Helper()
	pkg, err := c.Get(pkgPath)
	if err != nil {
		t.Fatalf("get %s: %v", pkgPath, err)
	}
	obj := pkg.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("%s.%s not found", pkgPath, structName)
	}
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		t.Fatalf("%s.%s is not a struct", pkgPath, structName)
	}
	for f := range st.Fields() {
		if f.Name() == field {
			return f.Type()
		}
	}
	t.Fatalf("field %s not found in %s.%s", field, pkgPath, structName)
	return nil
}

// TestSharedUniverseAcrossSeparateLoads is the regression guard for the batch
// master: two packages decoded by SEPARATE LoadWithCandidates calls (each doing
// its own packages.Load) must still land in one type universe, so a type they
// both reference is identical. This fails under a NeedTypes-per-call loader,
// where every packages.Load builds an isolated universe.
func TestSharedUniverseAcrossSeparateLoads(t *testing.T) {
	c := NewCache(".", "")

	if err := c.Load([]string{"net/http"}); err != nil {
		t.Fatalf("load net/http: %v", err)
	}
	if err := c.Load([]string{"archive/zip"}); err != nil {
		t.Fatalf("load archive/zip: %v", err)
	}

	httpTime := structFieldType(t, c, "net/http", "Cookie", "Expires")
	zipTime := structFieldType(t, c, "archive/zip", "FileHeader", "Modified")
	if !types.Identical(httpTime, zipTime) {
		t.Fatalf("time.Time from separate loads not identical: %s vs %s", httpTime, zipTime)
	}
}

// TestDirectLoadCompletesTransitiveStub guards the completeness-upgrade path:
// a package first pulled in as an incomplete transitive dependency must be
// re-decoded in place (same map entry) and become complete when requested
// directly, since the resolver queries arbitrary symbols in it.
func TestDirectLoadCompletesTransitiveStub(t *testing.T) {
	c := NewCache(".", "")
	if err := c.Load([]string{"net/http"}); err != nil {
		t.Fatalf("load net/http: %v", err)
	}

	stub := c.packages["time"]
	if stub == nil {
		t.Fatal("time not pulled in transitively by net/http")
	}

	if err := c.Load([]string{"time"}); err != nil {
		t.Fatalf("load time: %v", err)
	}
	got := c.packages["time"]
	if got != stub {
		t.Fatal("time entry replaced instead of upgraded in place")
	}
	if !got.Complete() {
		t.Fatal("time still incomplete after direct load")
	}
	if got.Scope().Lookup("Parse") == nil {
		t.Fatal("time.Parse not found after completion")
	}
}

// TestLoadSkipsCompleteCachedPaths keeps the incremental optimization: a
// complete package already in the cache is not re-decoded on a second load.
func TestLoadSkipsCompleteCachedPaths(t *testing.T) {
	c := NewCache(".", "")
	if err := c.Load([]string{"time"}); err != nil {
		t.Fatalf("load time: %v", err)
	}
	first := c.packages["time"]
	if first == nil || !first.Complete() {
		t.Fatalf("time not loaded complete on first Load")
	}

	if err := c.Load([]string{"time"}); err != nil {
		t.Fatalf("reload time: %v", err)
	}
	if c.packages["time"] != first {
		t.Fatal("complete cached package was re-decoded")
	}
}
