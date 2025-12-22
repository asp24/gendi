package main

import "fmt"

//go:generate go run github.com/asp24/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	fmt.Println("advanced example")
}
