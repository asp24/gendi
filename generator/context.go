package generator

import (
	di "github.com/asp24/gendi"
)

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	decoratorsByBase  map[string][]*serviceDef
	baseByDecorator   map[string]string
	loader            *TypeLoader
	imports           *ImportManager
	outputPkgPath     string
	containerName     string
	cfg               *di.Config
	paramGetters      map[string]string
}
