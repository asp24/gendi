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
