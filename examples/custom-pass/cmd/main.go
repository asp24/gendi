package main

import (
	"fmt"
	"os"
)

//go:generate go run ../tools/gendi/main.go --config=gendi.yaml --out=. --pkg=main

func main() {
	container := NewContainer(nil)

	server, err := container.GetServer()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	server.Start()
}
