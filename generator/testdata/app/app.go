package app

import "time"

type A struct{}

type B struct {
	A *A
}

type C struct{}

func NewA() *A {
	return &A{}
}

func NewB(a *A) *B {
	return &B{A: a}
}

func NewC() *C {
	return &C{}
}

type Logger struct {
	Prefix string
}

func NewLogger(prefix string) *Logger {
	return &Logger{Prefix: prefix}
}

type Timer struct {
	Delay time.Duration
}

func NewTimer(delay time.Duration) *Timer {
	return &Timer{Delay: delay}
}

type Service interface {
	Value() string
}

type BaseService struct{}

func NewServiceBase() Service {
	return &BaseService{}
}

func NewServiceBaseConcrete() *BaseService {
	return &BaseService{}
}

func (s *BaseService) Value() string { return "base" }

type DecoratorA struct {
	inner Service
}

func NewServiceDecoratorA(inner Service) Service {
	return &DecoratorA{inner: inner}
}

func NewServiceDecoratorAConcrete(inner Service) *DecoratorA {
	return &DecoratorA{inner: inner}
}

func (d *DecoratorA) Value() string { return d.inner.Value() }

type DecoratorB struct {
	inner Service
}

func NewServiceDecoratorB(inner Service) Service {
	return &DecoratorB{inner: inner}
}

func (d *DecoratorB) Value() string { return d.inner.Value() }

type Consumer struct {
	Svc Service
}

func NewConsumer(svc Service) *Consumer {
	return &Consumer{Svc: svc}
}

type InterfaceConsumer struct {
	Items []interface{}
}

func NewInterfaceConsumer(items []interface{}) *InterfaceConsumer {
	return &InterfaceConsumer{Items: items}
}
