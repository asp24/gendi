package main

import "fmt"

func main() {
	container := NewContainer(nil)
	app, err := container.GetApp()
	if err != nil {
		panic(err)
	}
	fmt.Println(app.Describe())
}
