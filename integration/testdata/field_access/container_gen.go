package main

type Container struct{}

func NewContainer(_ any) *Container {
	return &Container{}
}

func (c *Container) GetServer() (*Server, error) {
	panic("implement me")
}
