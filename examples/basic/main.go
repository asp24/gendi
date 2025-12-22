package main

import (
	"github.com/asp24/gendi/examples/basic/internal/di"
)

//go:generate go run github.com/asp24/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	container := &di.Container{}
	svc, err := container.GetService()
	if err != nil {
		panic(err)
	}

	svc.Logger.Log("Hello, World!")
}
