package main

import "fmt"

//go:generate di-gen --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	fmt.Println("advanced example")
}
