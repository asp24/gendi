package generator

import (
	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
)

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	tags              map[string]*ir.Tag
	loader            *typeres.Resolver
	imports           *ImportManager
	outputPkgPath     string
	containerName     string
	paramGetters      map[string]string
	nameGen           *nameGenerator
}
