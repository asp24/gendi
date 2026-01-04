package generator

// Renderer contains tools for rendering generated code.
type Renderer struct {
	imports       *ImportManager
	nameGen       *nameGenerator
	containerName string
}

// NewRenderer creates a new Renderer.
func NewRenderer(imports *ImportManager, nameGen *nameGenerator, containerName string) *Renderer {
	return &Renderer{
		imports:       imports,
		nameGen:       nameGen,
		containerName: containerName,
	}
}
