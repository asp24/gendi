package main

import (
	"fmt"

	"github.com/gendi-org/gendi/examples/advanced/internal/di"
)

//go:generate go run github.com/gendi-org/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	container := di.NewContainer(di.DefaultContainerParameters)
	_, err := container.GetHandler()
	if err != nil {
		panic(err)
	}

	fmt.Println("advanced example")
}
