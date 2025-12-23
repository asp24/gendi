package generator

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
)

type serviceDef struct {
	id                 string
	cfg                *di.Service
	typeName           types.Type
	constructor        constructorDef
	getterName         string
	privateGetterName  string
	public             bool
	shared             bool
	canError           bool
	decorates          string
	decorationPriority int
	isDecorator        bool
	aliasTarget        string
}

// IsAlias returns true if this service is an alias to another service.
func (s *serviceDef) IsAlias() bool {
	return s.cfg.Alias != ""
}

// HasConstructor returns true if this service defines a constructor.
func (s *serviceDef) HasConstructor() bool {
	return s.constructor.kind != ""
}

// Dependencies returns the service IDs this service depends on.
func (s *serviceDef) Dependencies(cfg *di.Config) ([]string, error) {
	return constructorDeps(s.id, s, cfg)
}

type constructorDef struct {
	kind         string // func|method
	funcObj      *types.Func
	methodObj    *types.Func
	methodRecvID string
	params       []types.Type
	result       types.Type
	returnsError bool
	argDefs      []di.Argument
}

func constructorDeps(id string, svc *serviceDef, cfg *di.Config) ([]string, error) {
	deps := []string{}
	if svc.aliasTarget != "" {
		return []string{svc.aliasTarget}, nil
	}
	cons := svc.constructor
	if cons.kind == "method" {
		deps = append(deps, cons.methodRecvID)
	}
	for _, arg := range cons.argDefs {
		switch arg.Kind {
		case di.ArgServiceRef:
			deps = append(deps, arg.Value)
		case di.ArgInner:
			if svc.decorates == "" {
				return nil, fmt.Errorf("service %q uses @.inner but is not a decorator", id)
			}
			deps = append(deps, svc.decorates)
		case di.ArgTagged:
			for sid, tagSvc := range cfg.Services {
				for _, t := range tagSvc.Tags {
					if t.Name == arg.Value {
						deps = append(deps, sid)
						break
					}
				}
			}
		}
	}
	return uniqueStrings(deps), nil
}

func buildDeps(id string, svc *serviceDef, cfg *di.Config) ([]string, error) {
	deps := []string{}
	if svc.aliasTarget != "" {
		return []string{svc.aliasTarget}, nil
	}
	cons := svc.constructor
	if cons.kind == "method" {
		deps = append(deps, cons.methodRecvID)
	}
	for _, arg := range cons.argDefs {
		switch arg.Kind {
		case di.ArgServiceRef:
			deps = append(deps, arg.Value)
		case di.ArgTagged:
			for sid, tagSvc := range cfg.Services {
				for _, t := range tagSvc.Tags {
					if t.Name == arg.Value {
						deps = append(deps, sid)
						break
					}
				}
			}
		case di.ArgInner:
			// inner is provided by decorator chain; its errors are handled there.
		}
	}
	return uniqueStrings(deps), nil
}

func getterType(svc *serviceDef, services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef) types.Type {
	if svc.aliasTarget != "" {
		if target := services[svc.aliasTarget]; target != nil {
			return getterType(target, services, decoratorsByBase)
		}
	}
	if svc.decorates != "" {
		return svc.typeName
	}
	if decs := decoratorsByBase[svc.id]; len(decs) > 0 {
		return decs[len(decs)-1].typeName
	}
	return svc.typeName
}
