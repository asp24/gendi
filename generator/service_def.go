package generator

import (
	"go/types"

	"github.com/asp24/gendi/ir"
)

type serviceDef struct {
	id                string
	typeName          types.Type
	declaredType      types.Type
	constructor       constructorDef
	getterName        string
	privateGetterName string
	public            bool
	shared            bool
	canError          bool
	aliasTarget       string
	tags              []*ir.ServiceTag
}

// IsAlias returns true if this service is an alias to another service.
func (s *serviceDef) IsAlias() bool {
	return s.aliasTarget != ""
}

// HasConstructor returns true if this service defines a constructor.
func (s *serviceDef) HasConstructor() bool {
	return s.constructor.kind != ""
}

func (s *serviceDef) GetterType() types.Type {
	return s.typeName
}

type constructorDef struct {
	kind         string // func|method
	funcObj      *types.Func
	methodRecvID string
	typeArgs     []types.Type // For generic functions
	params       []types.Type
	result       types.Type
	returnsError bool
	argDefs      []*ir.Argument
}
