package main

import (
	"fmt"
)

type Service interface {
	Process() string
}

type BaseService struct{}

func NewBaseService() Service { return &BaseService{} }

func (s *BaseService) Process() string { return "base" }

type LoggingDecorator struct{ inner Service }

func NewLoggingDecorator(inner Service) Service {
	return &LoggingDecorator{inner: inner}
}

func (d *LoggingDecorator) Process() string {
	return "log(" + d.inner.Process() + ")"
}

type MetricsDecorator struct{ inner Service }

func NewMetricsDecorator(inner Service) Service {
	return &MetricsDecorator{inner: inner}
}

func (d *MetricsDecorator) Process() string {
	return "metrics(" + d.inner.Process() + ")"
}

type CacheDecorator struct{ inner Service }

func NewCacheDecorator(inner Service) Service {
	return &CacheDecorator{inner: inner}
}

func (d *CacheDecorator) Process() string {
	return "cache(" + d.inner.Process() + ")"
}

type App struct {
	service Service
}

func NewApp(service Service) *App {
	return &App{service: service}
}

func (a *App) Run() {
	// Should wrap as: cache(metrics(log(base)))
	fmt.Println(a.service.Process())
}
