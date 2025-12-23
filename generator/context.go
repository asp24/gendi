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
	hasShared         bool
	buildCanError     map[string]bool
	getterCanError    map[string]bool
	cfg               *di.Config
	paramGetters      map[string]string
}
