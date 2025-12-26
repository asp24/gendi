package generator

import (
	"go/types"

	"github.com/asp24/gendi/ir"
)

type serviceDef struct {
	id                 string
	typeName           types.Type
	declaredType       types.Type
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
	tags               []*ir.ServiceTag
}

// IsAlias returns true if this service is an alias to another service.
func (s *serviceDef) IsAlias() bool {
	return s.aliasTarget != ""
}

// HasConstructor returns true if this service defines a constructor.
func (s *serviceDef) HasConstructor() bool {
	return s.constructor.kind != ""
}

type constructorDef struct {
	kind         string // func|method
	funcObj      *types.Func
	methodObj    *types.Func
	methodRecvID string
	typeArgs     []types.Type // For generic functions
	params       []types.Type
	result       types.Type
	returnsError bool
	argDefs      []*ir.Argument
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
