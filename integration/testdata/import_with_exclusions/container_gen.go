package main

type Container struct{}

func NewContainer(_ any) *Container {
	return &Container{}
}

func (c *Container) GetService() (*Service, error) {
	panic("implement me")
}

func (c *Container) MustService() *Service {
	panic("implement me")
}
