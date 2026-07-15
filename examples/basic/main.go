package main

import (
	"github.com/gendi-org/gendi/examples/basic/internal/di"
)

//go:generate go run github.com/gendi-org/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	container := di.NewContainer(nil)
	svc, err := container.GetService()
	if err != nil {
		panic(err)
	}

	svc.Logger.Log("Hello, World!")
}
