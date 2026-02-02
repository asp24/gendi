package main

import "fmt"

func (a *App) Run() {
	fmt.Printf("%s from %s\n", a.message, a.env)
}

func main() {
	container := NewContainer(nil)
	app, err := container.GetApp()
	if err != nil {
		panic(err)
	}
	app.Run()
}
