package generator

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	paramGetters      map[string]string
}
