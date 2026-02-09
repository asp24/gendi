package generator

type GenContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	paramGetters      map[string]string
}
