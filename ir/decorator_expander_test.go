package ir

import (
	"go/types"
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestDecoratorExpanderSingleDecorator(t *testing.T) {
	base := &Service{
		ID:          "svc",
		Type:        types.Typ[types.Int],
		Public:      true,
		Shared:      false,
		Constructor: &Constructor{Kind: FuncConstructor},
	}
	dec := &Service{
		ID:          "svc.decorator",
		Type:        types.Typ[types.String],
		Shared:      true,
		Constructor: &Constructor{Kind: FuncConstructor, Args: []*Argument{{Kind: InnerArg}}},
	}

	ctx := &buildContext{
		cfg: &di.Config{
			Services: map[string]di.Service{
				base.ID: {},
				dec.ID: {
					Decorates:          base.ID,
					DecorationPriority: 10,
				},
			},
		},
		services: map[string]*Service{
			base.ID: base,
			dec.ID:  dec,
		},
		order: []string{base.ID, dec.ID},
	}

	resolver := &decoratorResolver{}
	if err := resolver.resolve(ctx); err != nil {
		t.Fatalf("expand failed: %v", err)
	}

	rawBase := findServiceByPrefix(t, ctx.services, "__decorator_base__svc")
	if rawBase == nil {
		t.Fatalf("expected raw base service to be created")
	}
	if rawBase.Public || rawBase.Shared {
		t.Fatalf("expected raw base to be non-public and non-shared")
	}
	if rawBase.Constructor == nil || rawBase.Constructor.Kind != FuncConstructor {
		t.Fatalf("expected raw base to keep original constructor")
	}

	if base.Alias != nil {
		t.Fatalf("expected base alias to be cleared")
	}
	if base.Type != dec.Type {
		t.Fatalf("expected base type to match decorator type")
	}
	if !base.Shared {
		t.Fatalf("expected base to become shared due to decorator")
	}

	if base.Constructor == nil || base.Constructor == dec.Constructor {
		t.Fatalf("expected base to clone decorator constructor")
	}
	if len(base.Constructor.Args) != 1 || base.Constructor.Args[0].Kind != ServiceRefArg {
		t.Fatalf("expected inner arg to be rewritten to service ref")
	}
	if base.Constructor.Args[0].Service != rawBase {
		t.Fatalf("expected inner arg to reference raw base")
	}

	if dec.Alias != base {
		t.Fatalf("expected decorator to become alias to base")
	}
	if dec.Constructor != nil {
		t.Fatalf("expected decorator fields to be cleared after expansion")
	}
	if dec.Type != base.Type {
		t.Fatalf("expected decorator type to match base after expansion")
	}
}

func TestDecoratorExpanderChainCreatesInternalServices(t *testing.T) {
	base := &Service{
		ID:          "svc",
		Type:        types.Typ[types.Int],
		Public:      true,
		Constructor: &Constructor{Kind: FuncConstructor},
	}
	decA := &Service{
		ID:          "svc.decA",
		Type:        types.Typ[types.String],
		Constructor: &Constructor{Kind: FuncConstructor, Args: []*Argument{{Kind: InnerArg}}},
	}
	decB := &Service{
		ID:          "svc.decB",
		Type:        types.Typ[types.String],
		Shared:      true,
		Constructor: &Constructor{Kind: FuncConstructor, Args: []*Argument{{Kind: InnerArg}}},
	}

	ctx := &buildContext{
		cfg: &di.Config{
			Services: map[string]di.Service{
				base.ID: {},
				decA.ID: {
					Decorates:          base.ID,
					DecorationPriority: 10,
				},
				decB.ID: {
					Decorates:          base.ID,
					DecorationPriority: 20,
				},
			},
		},
		services: map[string]*Service{
			base.ID: base,
			decA.ID: decA,
			decB.ID: decB,
		},
		order: []string{base.ID, decA.ID, decB.ID},
	}

	resolver := &decoratorResolver{}
	if err := resolver.resolve(ctx); err != nil {
		t.Fatalf("expand failed: %v", err)
	}

	rawBase := findServiceByPrefix(t, ctx.services, "__decorator_base__svc")
	internalDecA := findServiceByPrefix(t, ctx.services, "__decorator_decorator__svc.decA")
	if rawBase == nil || internalDecA == nil {
		t.Fatalf("expected raw base and internal decorator services to be created")
	}
	if internalDecA.Shared {
		t.Fatalf("expected internal decorator to be non-shared")
	}

	if base.Constructor == nil || len(base.Constructor.Args) != 1 {
		t.Fatalf("expected base constructor to be replaced by outer decorator")
	}
	if base.Constructor.Args[0].Kind != ServiceRefArg || base.Constructor.Args[0].Service != internalDecA {
		t.Fatalf("expected outer decorator to reference internal decorator")
	}
	if internalDecA.Constructor == nil || len(internalDecA.Constructor.Args) != 1 {
		t.Fatalf("expected internal decorator to keep constructor")
	}
	if internalDecA.Constructor.Args[0].Kind != ServiceRefArg || internalDecA.Constructor.Args[0].Service != rawBase {
		t.Fatalf("expected internal decorator to reference raw base")
	}

	if !base.Shared {
		t.Fatalf("expected base to become shared due to decorator chain")
	}
	if decA.Alias != base || decB.Alias != base {
		t.Fatalf("expected decorators to become aliases to base")
	}
}

func TestDecoratorExpanderInnerArgOutsideDecorator(t *testing.T) {
	svc := &Service{
		ID:          "svc",
		Type:        types.Typ[types.Int],
		Public:      true,
		Constructor: &Constructor{Kind: FuncConstructor, Args: []*Argument{{Kind: InnerArg}}},
	}

	resolver := &decoratorResolver{}
	err := resolver.validateInnerArgs(map[string]*Service{svc.ID: svc})
	if err == nil || !strings.Contains(err.Error(), "@.inner used outside decorator") {
		t.Fatalf("expected inner arg error, got: %v", err)
	}
}

func TestDecoratorExpanderDecoratorCannotBeDecorated(t *testing.T) {
	base := &Service{ID: "base"}
	other := &Service{ID: "other"}
	dec := &Service{ID: "dec"}

	ctx := &buildContext{
		cfg: &di.Config{
			Services: map[string]di.Service{
				base.ID: {
					Decorates: "other",
				},
				dec.ID: {
					Decorates: base.ID,
				},
				other.ID: {},
			},
		},
		services: map[string]*Service{
			"base":  base,
			"dec":   dec,
			"other": other,
		},
		order: []string{"base", "dec", "other"},
	}

	resolver := &decoratorResolver{}
	err := resolver.resolve(ctx)
	if err == nil || !strings.Contains(err.Error(), "cannot be decorated") {
		t.Fatalf("expected decorated decorator error, got: %v", err)
	}
}

func findServiceByPrefix(t *testing.T, services map[string]*Service, prefix string) *Service {
	t.Helper()
	var found *Service
	for id, svc := range services {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		if found != nil {
			t.Fatalf("expected one service with prefix %q, found multiple", prefix)
		}
		found = svc
	}
	return found
}
