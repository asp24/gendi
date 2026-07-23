package main

import "fmt"

type Greeter struct{ name string }

func NewGreeter(name string) Greeter { return Greeter{name: name} }

func (g Greeter) Name() string { return g.name }

type App struct{ greeters []Greeter }

func NewApp(greeters ...Greeter) *App { return &App{greeters: greeters} }

func (a *App) Run() {
	for _, g := range a.greeters {
		fmt.Println(g.Name())
	}
}
