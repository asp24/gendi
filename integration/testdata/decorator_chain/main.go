package main

import "fmt"

func (a *App) Run() {
	// Should wrap as: cache(metrics(log(base)))
	fmt.Println(a.service.Process())
}

func main() {
	container := NewContainer(nil)
	app, err := container.GetApp()
	if err != nil {
		panic(err)
	}
	app.Run()
}
