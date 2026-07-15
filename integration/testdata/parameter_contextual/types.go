package main

type App struct {
	s string
	n int
}

func NewApp(s string, n int) *App {
	return &App{s: s, n: n}
}
