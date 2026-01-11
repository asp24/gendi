package generator

import (
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestMustGettersGenerated(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"foo": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewA",
				},
				Public: true,
			},
			"bar": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewC",
				},
				Public: true,
			},
			"internal": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
				Public: false, // not public
			},
		},
	}

	gen := New(testOptions(t))
	code, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that onMustCallFailed field is present
	if !strings.Contains(codeStr, "onMustCallFailed func(serviceName string, err error)") {
		t.Errorf("expected onMustCallFailed field in Container struct")
	}

	// Check that WithContainerErrorHandler is generated
	if !strings.Contains(codeStr, "func WithContainerErrorHandler(handler func(serviceName string, err error)) ContainerOption") {
		t.Errorf("expected WithContainerErrorHandler function")
	}

	// Check that NewContainer accepts options
	if !strings.Contains(codeStr, "func NewContainer(params parameters.Provider, opts ...ContainerOption)") {
		t.Errorf("expected NewContainer to accept options")
	}

	// Check that Must* methods are generated for public services
	if !strings.Contains(codeStr, "func (c *Container) MustFoo()") {
		t.Errorf("expected MustFoo method")
	}
	if !strings.Contains(codeStr, "func (c *Container) MustBar()") {
		t.Errorf("expected MustBar method")
	}

	// Check that Must* methods are NOT generated for private services
	if strings.Contains(codeStr, "func (c *Container) MustInternal()") {
		t.Errorf("unexpected MustInternal method for private service")
	}

	// Check that Must* methods call onMustCallFailed callback
	if !strings.Contains(codeStr, "c.onMustCallFailed(") {
		t.Errorf("expected Must methods to call onMustCallFailed callback")
	}

	// Check that Must* methods panic after callback
	if !strings.Contains(codeStr, `panic(err)`) {
		t.Errorf("expected Must methods to panic after onMustCallFailed")
	}
}
