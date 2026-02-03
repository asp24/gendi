package main

type Container struct{}

func NewContainer() *Container {
	return &Container{}
}

func (c *Container) GetApp() (*App, error) {
	panic("implement me")
}
