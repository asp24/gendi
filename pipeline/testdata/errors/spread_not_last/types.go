package main

type Handler interface{}

type App struct{}

func NewApp(name string, handlers ...Handler) *App {
	return &App{}
}
