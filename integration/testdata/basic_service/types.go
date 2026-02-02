package main

type Greeter struct {
	message string
}

func NewGreeter(msg string) *Greeter {
	return &Greeter{message: msg}
}

func (g *Greeter) Greet() string {
	return g.message
}
