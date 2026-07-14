package typeres

import "testing"

func TestComputeOutputPkgPath(t *testing.T) {
	got, err := ComputeOutputPkgPath("example.com/m", "/mod", "/mod/internal/di")
	if err != nil || got != "example.com/m/internal/di" {
		t.Errorf("got %q, %v", got, err)
	}

	got, err = ComputeOutputPkgPath("example.com/m", "/mod", "/mod/di/container_gen.go")
	if err != nil || got != "example.com/m/di" {
		t.Errorf("got %q, %v", got, err)
	}

	got, err = ComputeOutputPkgPath("example.com/m", "/mod", "/mod")
	if err != nil || got != "example.com/m" {
		t.Errorf("got %q, %v", got, err)
	}

	// Output outside the module cannot have a valid import path.
	if _, err := ComputeOutputPkgPath("example.com/m", "/mod", "/elsewhere/di"); err == nil {
		t.Error("expected error for output directory outside the module root")
	}
}
