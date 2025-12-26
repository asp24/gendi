package main

import (
	"log"
)

//go:generate go run ../tools/gendi/main.go --config=gendi.yaml --out=. --pkg=main

func main() {
	container := NewContainer(nil)

	server, err := container.GetServer()
	if err != nil {
		log.Fatal(err)
	}

	server.Start()
}
