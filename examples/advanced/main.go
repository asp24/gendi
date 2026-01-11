package main

import (
	"fmt"

	"github.com/asp24/gendi/examples/advanced/internal/di"
)

//go:generate go run github.com/asp24/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	container := di.NewContainer(di.DefaultContainerParameters)
	_, err := container.GetHandler()
	if err != nil {
		panic(err)
	}

	fmt.Println("advanced example")
}
