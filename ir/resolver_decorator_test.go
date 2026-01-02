package ir

import (
	"go/types"
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestDecoratorResolverSingleDecorator(t *testing.T) {
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

	rawBase := ctx.services["svc.decorator.inner"]
	if rawBase == nil {
		t.Fatalf("expected raw base service to be created")
	}
	if rawBase.Constructor == nil || rawBase.Constructor.Kind != FuncConstructor {
		t.Fatalf("expected raw base to keep original constructor")
	}

	if base.Alias != dec {
		t.Fatalf("expected base to become alias to decorator")
	}
	if base.Constructor != nil {
		t.Fatalf("expected base constructor to be cleared")
	}
	if base.Type != dec.Type {
		t.Fatalf("expected base type to match decorator type")
	}
	if !base.Shared {
		t.Fatalf("expected base to become shared due to decorator")
	}

	if dec.Alias != nil {
		t.Fatalf("expected decorator to remain a concrete service")
	}
	if dec.Constructor == nil || len(dec.Constructor.Args) != 1 {
		t.Fatalf("expected decorator constructor to be retained and rewritten")
	}
	if dec.Constructor.Args[0].Kind != ServiceRefArg {
		t.Fatalf("expected inner arg to be rewritten to service ref")
	}
	if dec.Constructor.Args[0].Service != rawBase {
		t.Fatalf("expected inner arg to reference raw base")
	}
}

func TestDecoratorResolverChainCreatesInternalServices(t *testing.T) {
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

	rawBase := ctx.services["svc.decA.inner"]
	if rawBase == nil {
		t.Fatalf("expected raw base service to be created")
	}

	if base.Alias != decB {
		t.Fatalf("expected base to become alias to outer decorator")
	}
	if base.Constructor != nil {
		t.Fatalf("expected base constructor to be cleared")
	}
	if !base.Shared {
		t.Fatalf("expected base to become shared due to decorator chain")
	}

	if decA.Alias != nil || decB.Alias != nil {
		t.Fatalf("expected decorators to remain concrete services")
	}
	if decA.Shared {
		t.Fatalf("expected inner decorator to remain non-shared")
	}
	if decB.Constructor == nil || len(decB.Constructor.Args) != 1 {
		t.Fatalf("expected outer decorator to keep constructor")
	}
	if decB.Constructor.Args[0].Kind != ServiceRefArg || decB.Constructor.Args[0].Service != decA {
		t.Fatalf("expected outer decorator to reference inner decorator")
	}
	if decA.Constructor == nil || len(decA.Constructor.Args) != 1 {
		t.Fatalf("expected inner decorator to keep constructor")
	}
	if decA.Constructor.Args[0].Kind != ServiceRefArg || decA.Constructor.Args[0].Service != rawBase {
		t.Fatalf("expected inner decorator to reference raw base")
	}
}

func TestDecoratorResolverInnerArgOutsideDecorator(t *testing.T) {
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

func TestDecoratorResolverDecoratorCannotBeDecorated(t *testing.T) {
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
