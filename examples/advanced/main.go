package main

import "fmt"

//go:generate di-gen --config=di.yaml --out=./internal/di --pkg=di

func main() {
	fmt.Println("advanced example")
}

