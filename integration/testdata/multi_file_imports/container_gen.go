package main

type Container struct{}

func NewContainer() *Container {
	return &Container{}
}

func (c *Container) GetProduct() (*Product, error) {
	panic("implement me")
}
