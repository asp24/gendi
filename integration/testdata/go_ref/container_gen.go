package main

type Container struct{}

func NewContainer(_ any) *Container {
	return &Container{}
}

func (c *Container) GetWriter() (*Writer, error) {
	panic("implement me")
}
