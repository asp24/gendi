package main

import "fmt"

type Plugin interface {
	Name() string
}

type PluginA struct{}

func NewPluginA() Plugin { return &PluginA{} }

func (p *PluginA) Name() string { return "A" }

type PluginB struct{}

func NewPluginB() Plugin { return &PluginB{} }

func (p *PluginB) Name() string { return "B" }

type App struct {
	plugins []Plugin
}

func NewApp(plugins []Plugin) *App {
	return &App{plugins: plugins}
}

func (a *App) Run() {
	for _, p := range a.plugins {
		fmt.Println(p.Name())
	}
}
