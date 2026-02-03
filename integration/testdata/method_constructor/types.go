package main

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(name string) *Product {
	return &Product{name: name}
}

type Product struct {
	name string
}

func (p *Product) Name() string {
	return p.name
}
