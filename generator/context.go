package generator

import "github.com/asp24/gendi/ir"

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	tags              map[string]*ir.Tag
	loader            *TypeLoader
	imports           *ImportManager
	outputPkgPath     string
	containerName     string
	paramGetters      map[string]string
	nameGen           *nameGenerator
}
