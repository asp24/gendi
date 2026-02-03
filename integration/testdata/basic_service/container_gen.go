package main

type Container struct{}

func NewContainer(_ any) *Container {
	return &Container{}
}

func (c *Container) GetGreeter() (*Greeter, error) {
	panic("implement me")
}
