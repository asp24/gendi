package main

type A struct{ Name string }

type B struct{ Name string }

func NewA() *A { return &A{Name: "a"} }

func NewB() *B { return &B{Name: "b"} }

type Consumer struct {
	a *A
	b *B
}

func NewConsumer(a *A, b *B) *Consumer { return &Consumer{a: a, b: b} }

func (c *Consumer) Info() string { return c.a.Name + " " + c.b.Name }
