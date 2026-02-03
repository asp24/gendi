package main

type Container struct{}

func NewContainer(_ any) *Container {
	return &Container{}
}

func (c *Container) GetProduct() (*Product, error) {
	panic("implement me")
}

func (c *Container) MustProduct() *Product {
	panic("implement me")
}
