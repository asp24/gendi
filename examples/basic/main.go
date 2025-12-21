package main

import (
	"github.com/asp24/gendi/examples/basic/internal/di"
)

//go:generate di-gen --config=di.yaml --out=./internal/di --pkg=di

func main() {
	container := &di.Container{}
	svc, err := container.GetService()
	if err != nil {
		panic(err)
	}

	svc.Logger.Log("Hello, World!")
}
