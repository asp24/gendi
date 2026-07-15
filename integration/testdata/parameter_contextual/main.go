package main

import "fmt"

func main() {
	container := NewContainer(nil)
	app, err := container.GetApp()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s %d\n", app.s, app.n)
}
