package generator

type GenContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
}
