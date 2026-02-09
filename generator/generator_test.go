package generator_test

import (
	"strings"
	"testing"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/generator"
	"github.com/asp24/gendi/pipeline"
	"github.com/asp24/gendi/yaml"
)

func testEmitOptions(t *testing.T) (pipeline.Options, generator.EmitOptions) {
	t.Helper()
	opts := pipeline.Options{Out: ".", Package: "di"}
	if err := opts.Finalize(); err != nil {
		t.Fatalf("finalize options: %v", err)
	}
	emitOpts := generator.EmitOptions{
		Package:       opts.Package,
		Container:     opts.Container,
		OutputPkgPath: opts.OutputPkgPath,
		BuildTags:     opts.BuildTags,
	}
	return opts, emitOpts
}

func generate(t *testing.T, cfg *di.Config) string {
	t.Helper()
	opts, emitOpts := testEmitOptions(t)
	compiled, err := pipeline.Build(cfg, opts.ModuleRoot)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	code, err := generator.Emit(compiled.Config, compiled.IR, compiled.TypeResolver, emitOpts)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	return string(code)
}

func generateErr(t *testing.T, cfg *di.Config) error {
	t.Helper()
	opts, emitOpts := testEmitOptions(t)
	compiled, err := pipeline.Build(cfg, opts.ModuleRoot)
	if err != nil {
		return err
	}
	_, err = generator.Emit(compiled.Config, compiled.IR, compiled.TypeResolver, emitOpts)
	return err
}

func TestRequiresPublicService(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"a": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewA",
				},
			},
		},
	}
	err := generateErr(t, cfg)
	if err == nil || !strings.Contains(err.Error(), "at least one public service or tag") {
		t.Fatalf("expected public service error, got %v", err)
	}
}

func TestReachabilityAndPublicGetters(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"a": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewA",
				},
			},
			"b": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewB",
					Args: []di.Argument{
						{Kind: di.ArgServiceRef, Value: "a"},
					},
				},
				Public: true,
			},
			"unused": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewC",
				},
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "GetB") {
		t.Fatalf("expected public getter for b")
	}
	if strings.Contains(out, "GetA") {
		t.Fatalf("unexpected public getter for private service a")
	}
	if !strings.Contains(out, "getA") {
		t.Fatalf("expected private getter for a")
	}
	if strings.Contains(out, "buildUnused") || strings.Contains(out, "getUnused") || strings.Contains(out, "GetUnused") {
		t.Fatalf("unexpected generation for unreachable service")
	}
	if strings.Contains(out, "svc_unused") {
		t.Fatalf("unexpected field for unreachable service")
	}
}

func TestPublicTagGetter(t *testing.T) {
	cfg := &di.Config{
		Tags: map[string]di.Tag{
			"svc.tag": {
				ElementType: "github.com/asp24/gendi/generator/testdata/app.Service",
				Public:      true,
			},
		},
		Services: map[string]di.Service{
			"base": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
				Tags: []di.ServiceTag{
					{Name: "svc.tag"},
				},
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "GetTaggedWithSvcTag") {
		t.Fatalf("expected public tag getter to be generated")
	}
	if !strings.Contains(out, "getTaggedWithSvcTag") {
		t.Fatalf("expected private tag getter to be generated")
	}
	if !strings.Contains(out, "[]app.Service") {
		t.Fatalf("expected tag getter to use declared element type")
	}
	if !strings.Contains(out, "getBase") {
		t.Fatalf("expected tagged service to be reachable")
	}
	if strings.Contains(out, "stdlib.MakeSlice") {
		t.Fatalf("expected stdlib.MakeSlice to be inlined")
	}
	if !strings.Contains(out, "[]app.Service{") {
		t.Fatalf("expected tag constructor to use slice literal")
	}
}

func TestTaggedInjectionConversion(t *testing.T) {
	cfg := &di.Config{
		Tags: map[string]di.Tag{
			"test.tag": {
				ElementType: "*github.com/asp24/gendi/generator/testdata/app.A",
			},
		},
		Services: map[string]di.Service{
			"a": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewA",
				},
				Tags: []di.ServiceTag{
					{Name: "test.tag"},
				},
			},
			"consumer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewInterfaceConsumer",
					Args: []di.Argument{
						{Kind: di.ArgTagged, Value: "test.tag"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "getTaggedWithTestTag") {
		t.Fatalf("expected private tag getter to be used")
	}
	if !strings.Contains(out, "var arg0_tagged_test_tag []interface{}") {
		t.Fatalf("expected tagged conversion destination variable")
	}
	if !strings.Contains(out, "arg0_tagged_test_tag = make([]interface{}, len(tagged_test_tag))") {
		t.Fatalf("expected tagged conversion slice allocation")
	}
	if !strings.Contains(out, "for idx, item := range tagged_test_tag") {
		t.Fatalf("expected tagged conversion loop")
	}
	if !strings.Contains(out, "arg0_tagged_test_tag[idx] = item") {
		t.Fatalf("expected tagged conversion assignment")
	}
}

func TestParameterProviderCodegen(t *testing.T) {
	cfg := &di.Config{
		Parameters: map[string]di.Parameter{
			"log_prefix": {
				Type:  "string",
				Value: di.NewStringLiteral("[app] "),
			},
		},
		Services: map[string]di.Service{
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgParam, Value: "log_prefix"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)
	if !strings.Contains(out, "NewContainer") {
		t.Fatalf("expected container constructor when parameters are present")
	}
	if !strings.Contains(out, "GetString(\"log_prefix\")") {
		t.Fatalf("expected parameter provider lookup in generated code")
	}
}

func TestDurationParameterCodegen(t *testing.T) {
	cfg := &di.Config{
		Parameters: map[string]di.Parameter{
			"timeout": {
				Type:  "time.Duration",
				Value: di.NewStringLiteral("1s"),
			},
		},
		Services: map[string]di.Service{
			"timer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewTimer",
					Args: []di.Argument{
						{Kind: di.ArgParam, Value: "timeout"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)
	if !strings.Contains(out, "GetDuration(\"timeout\")") {
		t.Fatalf("expected duration parameter lookup")
	}
}

func TestNullLiteralArgument(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"b": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewB",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewNullLiteral()},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)
	if !strings.Contains(out, "NewB(nil)") {
		t.Fatalf("expected nil literal for null argument")
	}
}

func TestServiceAliasCodegen(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgParam, Value: "log_prefix"},
					},
				},
				Public: true,
			},
			"logger.alias": {
				Alias:  "logger",
				Public: true,
			},
		},
		Parameters: map[string]di.Parameter{
			"log_prefix": {
				Type:  "string",
				Value: di.NewStringLiteral("[app] "),
			},
		},
	}

	out := generate(t, cfg)
	if !strings.Contains(out, "GetLoggerAlias") {
		t.Fatalf("expected alias public getter")
	}
	if strings.Contains(out, "buildLoggerAlias") {
		t.Fatalf("unexpected build function for alias")
	}
	if !strings.Contains(out, "return c.getLogger()") {
		t.Fatalf("expected alias getter to forward to target")
	}
}

func TestDecoratorPrivateGetterGeneratedForChain(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
				Public: true,
				Shared: true,
			},
			"svc.decoratorA": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
				Shared:             true,
			},
			"svc.decoratorB": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorB",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 20,
				Shared:             true,
			},
		},
	}

	out := generate(t, cfg)
	if !strings.Contains(out, "getSvcDecoratorB") {
		t.Fatalf("expected private getter for outer decorator")
	}
	if !strings.Contains(out, "getSvcDecoratorA") {
		t.Fatalf("expected private getter for inner decorator")
	}
	if !strings.Contains(out, "getSvcDecoratorAInner") {
		t.Fatalf("expected private getter for raw base")
	}
	// Note: After DecoratorPass refactoring, decoratorB (the outermost decorator)
	// is public via alias and gets a storage field. This is expected behavior.
	if !strings.Contains(out, "svc_svc_decoratorB") {
		t.Fatalf("expected storage field for outer decorator (public via alias)")
	}
	// Inner decorator A should be optimized away (used only by B)
	if strings.Contains(out, "svc_svc_decoratorA ") {
		t.Fatalf("unexpected storage field for inner decorator (should be non-shared)")
	}
}

func TestDecoratorPrivateGetterGeneratedWhenReferenced(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
			},
			"svc.decoratorA": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
			},
			"consumer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewConsumer",
					Args: []di.Argument{
						{Kind: di.ArgServiceRef, Value: "svc.decoratorA"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)
	if !strings.Contains(out, "getSvcDecoratorA") {
		t.Fatalf("expected private getter for referenced decorator")
	}
	if strings.Contains(out, "svc_svc_decoratorA ") {
		t.Fatalf("unexpected shared field for referenced decorator")
	}
}

func TestServiceTypeAssignableOverride(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"svc": {
				Type: "github.com/asp24/gendi/generator/testdata/app.Service",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBaseConcrete",
				},
				Public: true,
			},
		},
	}

	if err := generateErr(t, cfg); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func TestDecoratorAssignableToDeclaredBaseType(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"svc": {
				Type: "github.com/asp24/gendi/generator/testdata/app.Service",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBaseConcrete",
				},
				Public: true,
			},
			"svc.decorator": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorAConcrete",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
			},
		},
	}

	if err := generateErr(t, cfg); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func TestGenericFunctionConstructor(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"events": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewChan[github.com/asp24/gendi/generator/testdata/generics.Event]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(100)},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewChan[generics.Event]") {
		t.Fatalf("expected generic function call with type arguments, got:\n%s", out)
	}
	if !strings.Contains(out, "chan generics.Event") {
		t.Fatalf("expected chan Event return type, got:\n%s", out)
	}
}

func TestGenericPoolConstructor(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"message_pool": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewPool[github.com/asp24/gendi/generator/testdata/generics.Message]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(10)},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewPool[generics.Message]") {
		t.Fatalf("expected generic function call with type arguments, got:\n%s", out)
	}
	if !strings.Contains(out, "*generics.Pool[generics.Message]") {
		t.Fatalf("expected *Pool[Message] return type, got:\n%s", out)
	}
}

func TestGenericSliceTypeArg(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"event_slice": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewSlice[[]github.com/asp24/gendi/generator/testdata/generics.Event]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(10)},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewSlice[[]generics.Event]") {
		t.Fatalf("expected generic function call with slice type argument, got:\n%s", out)
	}
}

func TestGenericMapTypeArgs(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"event_map": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewMap[string, github.com/asp24/gendi/generator/testdata/generics.Event]",
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewMap[string, generics.Event]") {
		t.Fatalf("expected generic function call with map type arguments, got:\n%s", out)
	}
	if !strings.Contains(out, "map[string]generics.Event") {
		t.Fatalf("expected map[string]Event return type, got:\n%s", out)
	}
}

func TestGenericChanTypeArg(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"chan_of_chans": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewChan[chan github.com/asp24/gendi/generator/testdata/generics.Event]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(5)},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewChan[chan generics.Event]") {
		t.Fatalf("expected generic function call with chan type argument, got:\n%s", out)
	}
	if !strings.Contains(out, "chan chan generics.Event") {
		t.Fatalf("expected chan chan Event return type, got:\n%s", out)
	}
}

func TestGenericFunctionWithoutTypeArgsError(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"events": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewChan",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(100)},
					},
				},
				Public: true,
			},
		},
	}

	err := generateErr(t, cfg)
	if err == nil {
		t.Fatal("expected error for generic function without type arguments")
	}
	if !strings.Contains(err.Error(), "generic function") || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected error about missing type arguments, got: %v", err)
	}
}

func TestGenericTypeWithTypeArgs(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"pool": {
				Type: "*github.com/asp24/gendi/generator/testdata/generics.Pool[github.com/asp24/gendi/generator/testdata/generics.Message]",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewPool[github.com/asp24/gendi/generator/testdata/generics.Message]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(10)},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "*generics.Pool[generics.Message]") {
		t.Fatalf("expected *Pool[Message] type, got:\n%s", out)
	}
}

func TestGenericTypeWithoutTypeArgsError(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"pool": {
				Type: "github.com/asp24/gendi/generator/testdata/generics.Pool",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewPool[github.com/asp24/gendi/generator/testdata/generics.Message]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(10)},
					},
				},
				Public: true,
			},
		},
	}

	err := generateErr(t, cfg)
	if err == nil {
		t.Fatal("expected error for generic type without type arguments")
	}
	if !strings.Contains(err.Error(), "generic type") || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected error about missing type arguments, got: %v", err)
	}
}

func TestNonGenericTypeWithTypeArgsError(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"event": {
				Type: "github.com/asp24/gendi/generator/testdata/generics.Event[string]",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/generics.NewChan[github.com/asp24/gendi/generator/testdata/generics.Event]",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: di.NewIntLiteral(10)},
					},
				},
				Public: true,
			},
		},
	}

	err := generateErr(t, cfg)
	if err == nil {
		t.Fatal("expected error for non-generic type with type arguments")
	}
	if !strings.Contains(err.Error(), "not generic") {
		t.Fatalf("expected error about type not being generic, got: %v", err)
	}
}

func TestSpreadWithServiceRef(t *testing.T) {
	cfg, err := yaml.LoadConfig("testdata/spread/service_ref.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewServer(") {
		t.Fatal("expected NewServer call")
	}
	if !strings.Contains(out, "...") {
		t.Fatal("expected spread operator ... in generated code")
	}
	if !strings.Contains(out, "getAllHandlers()") {
		t.Fatal("expected getAllHandlers call")
	}
}

func TestSpreadWithTagged(t *testing.T) {
	cfg, err := yaml.LoadConfig("testdata/spread/tagged.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewServer(") {
		t.Fatal("expected NewServer call")
	}
	if !strings.Contains(out, "...") {
		t.Fatal("expected spread operator ... in generated code")
	}
	if !strings.Contains(out, "handler") {
		t.Fatal("expected handler services to be generated")
	}
}

func TestGoRefArgument(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"writer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewWriter",
					Args: []di.Argument{
						{Kind: di.ArgGoRef, Value: "os.Stdout"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "os.Stdout") {
		t.Fatalf("expected os.Stdout in generated code, got:\n%s", out)
	}
	if !strings.Contains(out, "NewWriter(os.Stdout)") {
		t.Fatalf("expected NewWriter(os.Stdout) call, got:\n%s", out)
	}
}

func TestGoRefArgumentWithPackageLevelVar(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgGoRef, Value: "github.com/asp24/gendi/generator/testdata/app.DefaultPrefix"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "app.DefaultPrefix") {
		t.Fatalf("expected app.DefaultPrefix in generated code, got:\n%s", out)
	}
	if !strings.Contains(out, "NewLogger(app.DefaultPrefix)") {
		t.Fatalf("expected NewLogger(app.DefaultPrefix) call, got:\n%s", out)
	}
}

func TestGoRefArgumentTypeMismatch(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgGoRef, Value: "os.Stdout"},
					},
				},
				Public: true,
			},
		},
	}

	err := generateErr(t, cfg)
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
	if !strings.Contains(err.Error(), "not assignable") {
		t.Fatalf("expected assignability error, got: %v", err)
	}
}

func TestFieldAccessOnService(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"config": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.LoadConfig",
				},
			},
			"server": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServerWithAddr",
					Args: []di.Argument{
						{Kind: di.ArgFieldAccessService, Value: "config.Host"},
						{Kind: di.ArgFieldAccessService, Value: "config.Port"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, ".Host") {
		t.Fatalf("expected .Host field access in generated code, got:\n%s", out)
	}
	if !strings.Contains(out, ".Port") {
		t.Fatalf("expected .Port field access in generated code, got:\n%s", out)
	}
}

func TestFieldAccessNested(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"config": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.LoadConfig",
				},
			},
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgFieldAccessService, Value: "config.Database.DSN"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, ".Database.DSN") {
		t.Fatalf("expected .Database.DSN field access in generated code, got:\n%s", out)
	}
}

func TestFieldAccessOnGoRef(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"timer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewTimer",
					Args: []di.Argument{
						{Kind: di.ArgFieldAccessGo, Value: "net/http.DefaultClient.Timeout"},
					},
				},
				Public: true,
			},
		},
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "http.DefaultClient.Timeout") {
		t.Fatalf("expected http.DefaultClient.Timeout in generated code, got:\n%s", out)
	}
}

func TestFieldAccessTypeMismatch(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"config": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.LoadConfig",
				},
			},
			"timer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewTimer",
					Args: []di.Argument{
						{Kind: di.ArgFieldAccessService, Value: "config.Host"},
					},
				},
				Public: true,
			},
		},
	}

	err := generateErr(t, cfg)
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
	if !strings.Contains(err.Error(), "not assignable") {
		t.Fatalf("expected assignability error, got: %v", err)
	}
}

func TestFieldAccessUnknownField(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"config": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.LoadConfig",
				},
			},
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgFieldAccessService, Value: "config.NonExistentField"},
					},
				},
				Public: true,
			},
		},
	}

	err := generateErr(t, cfg)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestSpreadWithMixedArgs(t *testing.T) {
	cfg, err := yaml.LoadConfig("testdata/spread/mixed_args.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	out := generate(t, cfg)

	if !strings.Contains(out, "NewServer(") {
		t.Fatal("expected NewServer call")
	}
	if !strings.Contains(out, "...") {
		t.Fatal("expected spread operator ... in generated code")
	}
	if !strings.Contains(out, "getHandlerA()") {
		t.Fatal("expected getHandlerA call for regular arg")
	}
	if !strings.Contains(out, "getMoreHandlers()") {
		t.Fatal("expected getMoreHandlers call for spread arg")
	}
}
