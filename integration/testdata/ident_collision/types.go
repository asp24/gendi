package main

type A struct{}

type B struct{}

func NewA() *A { return &A{} }

func NewB() *B { return &B{} }
