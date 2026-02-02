package main

import "fmt"

func main() {
	container := NewContainer(nil)
	greeter, err := container.GetGreeter()
	if err != nil {
		panic(err)
	}
	fmt.Println(greeter.Greet())
}
