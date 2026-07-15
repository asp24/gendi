package main

import (
	"context"
	"time"

	"github.com/gendi-org/gendi/examples/decorator/internal/di"
)

//go:generate go run github.com/gendi-org/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	container := &di.Container{}
	jobs, err := container.GetTaggedWithJob()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	for _, job := range jobs {
		if err := job.Run(ctx); err != nil {
			panic(err)
		}
	}
}
