package generator

import (
	"go/types"

	"github.com/asp24/gendi/ir"
)

type serviceDef struct {
	id                string
	typeName          types.Type
	constructor       constructorDef
	privateGetterName string
	public            bool
	shared            bool
	canError          bool
	aliasTarget       string
}

// IsAlias returns true if this service is an alias to another service.
func (s *serviceDef) IsAlias() bool {
	return s.aliasTarget != ""
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
