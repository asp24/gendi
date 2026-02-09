package pipeline

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
	"github.com/asp24/gendi/xmaps"
)

// Output is the compiled result of the config pipeline, ready for code generation.
type Output struct {
	Config       *di.Config
	IR           *ir.Container
	TypeResolver *typeres.Resolver
}

// Build compiles a DI config through internal passes, type loading, and IR building.
func Build(cfg *di.Config, moduleRoot string) (*Output, error) {
	// Apply internal passes (idempotent - decorators already expanded will be skipped)
	cfg, err := di.ApplyInternalPasses(cfg)
	if err != nil {
		return nil, err
	}

	// Re-populate Packages fields after passes may have added/modified services.
	refreshPackages(cfg)

	paths := collectPackagePaths(cfg)
	typeResolver := typeres.NewResolver(moduleRoot)
	if err := typeResolver.LoadPackages(paths); err != nil {
		return nil, err
	}

	irBuilder := ir.NewBuilder(typeResolver)
	irContainer, err := irBuilder.Build(cfg)
	if err != nil {
		return nil, err
	}

	return &Output{
		Config:       cfg,
		IR:           irContainer,
		TypeResolver: typeResolver,
	}, nil
}

// refreshPackages re-populates the Packages field on all config structs.
// This is needed after compiler passes that may add or modify services.
func refreshPackages(cfg *di.Config) {
	for name, param := range cfg.Parameters {
		param.Packages = typeres.CollectTypePackages(param.Type)
		cfg.Parameters[name] = param
	}
	for name, tag := range cfg.Tags {
		tag.Packages = typeres.CollectTypePackages(tag.ElementType)
		cfg.Tags[name] = tag
	}
	for name, svc := range cfg.Services {
		svc.Packages = typeres.CollectTypePackages(svc.Type)
		svc.Constructor.Packages = typeres.CollectFuncPackages(svc.Constructor.Func)
		for i, arg := range svc.Constructor.Args {
			switch arg.Kind {
			case di.ArgGoRef:
				arg.Packages = typeres.CollectGoRefPackages(arg.Value)
			case di.ArgFieldAccessGo:
				arg.Packages = typeres.CollectFieldAccessGoPackages(arg.Value)
			}
			svc.Constructor.Args[i] = arg
		}
		cfg.Services[name] = svc
	}
}

func collectPackagePaths(cfg *di.Config) []string {
	seen := map[string]bool{}
	addAll := func(pkgs []string) {
		for _, p := range pkgs {
			if p != "" {
				seen[p] = true
			}
		}
	}

	for _, svc := range cfg.Services {
		addAll(svc.Packages)
		addAll(svc.Constructor.Packages)
		for _, arg := range svc.Constructor.Args {
			addAll(arg.Packages)
		}
	}
	for _, param := range cfg.Parameters {
		addAll(param.Packages)
	}
	for _, tag := range cfg.Tags {
		addAll(tag.Packages)
	}

	if hasTagsOrTaggedArgs(cfg) {
		seen["github.com/asp24/gendi/stdlib"] = true
	}

	return xmaps.OrderedKeys(seen)
}

func hasTagsOrTaggedArgs(cfg *di.Config) bool {
	if len(cfg.Tags) > 0 {
		return true
	}
	for _, svc := range cfg.Services {
		if len(svc.Tags) > 0 {
			return true
		}
		for _, arg := range svc.Constructor.Args {
			if arg.Kind == di.ArgTagged {
				return true
			}
		}
	}
	return false
}
