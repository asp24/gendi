package main

import "fmt"

func (a *App) Run() {
	fmt.Printf("%s: %d handlers\n", a.name, len(a.handlers))
	for _, h := range a.handlers {
		fmt.Println(h.Name())
	}
}

func main() {
	container := NewContainer(nil)
	app, err := container.GetApp()
	if err != nil {
		panic(err)
	}
	app.Run()
}
