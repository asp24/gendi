package app

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
