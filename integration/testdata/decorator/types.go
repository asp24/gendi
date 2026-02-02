package main

import "fmt"

type Service interface {
	Process() string
}

type BaseService struct{}

func NewService() Service {
	return &BaseService{}
}

func (s *BaseService) Process() string {
	return "base"
}

type Decorator struct {
	inner Service
}

func NewDecorator(inner Service) Service {
	return &Decorator{inner: inner}
}

func (d *Decorator) Process() string {
	return "decorated(" + d.inner.Process() + ")"
}

type App struct {
	service Service
}

func NewApp(service Service) *App {
	return &App{service: service}
}

func (a *App) Run() {
	fmt.Println(a.service.Process())
}
