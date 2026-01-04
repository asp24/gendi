package generator

import (
	"github.com/asp24/gendi/ir"
)

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	tags              map[string]*ir.Tag
	outputPkgPath     string
	paramGetters      map[string]string
}
