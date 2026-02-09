package main

import "fmt"

func main() {
	container := NewContainer(nil)
	server, err := container.GetServer()
	if err != nil {
		panic(err)
	}
	fmt.Println(server.Info())
}
