package generator

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	outputPkgPath     string
	paramGetters      map[string]string
}
