package generator

// Renderer contains tools for rendering generated code.
type Renderer struct {
	imports       *ImportManager
	ident         *identGenerator
	getters       *getterRegistry
	containerName string
}

// NewRenderer creates a new Renderer.
func NewRenderer(imports *ImportManager, ident *identGenerator, getters *getterRegistry, containerName string) *Renderer {
	return &Renderer{
		imports:       imports,
		ident:         ident,
		getters:       getters,
		containerName: containerName,
	}
}
