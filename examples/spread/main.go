package main

import (
	"fmt"

	"github.com/asp24/gendi/examples/spread/internal/di"
)

//go:generate go run github.com/asp24/gendi/cmd/gendi --config=gendi.yaml --out=./internal/di --pkg=di

func main() {
	container := di.NewContainer(nil)

	fmt.Println("=== Spread Operator Example ===")
	fmt.Println("\nThis example demonstrates three use cases of the spread operator:")
	fmt.Println("1. Spreading tagged injection into variadic parameters")
	fmt.Println("2. Spreading service references into variadic parameters")
	fmt.Println("3. Mixing regular arguments with spread")

	// Example 1: Spread tagged injection
	fmt.Println("\n--- Example 1: Spread Tagged Injection ---")
	fmt.Println("Config: !spread:!tagged:handler")
	serverTagged, err := container.GetServerTagged()
	if err != nil {
		panic(err)
	}
	serverTagged.ShowHandlers()
	serverTagged.HandleRequest("/users")

	// Example 2: Spread service reference
	fmt.Println("\n--- Example 2: Spread Service Reference ---")
	fmt.Println("Config: !spread:@all_handlers")
	serverRef, err := container.GetServerRef()
	if err != nil {
		panic(err)
	}
	serverRef.ShowHandlers()
	serverRef.HandleRequest("/products")

	// Example 3: Mixed arguments with spread
	fmt.Println("\n--- Example 3: Mixed Arguments with Spread ---")
	fmt.Println("Config: prefix parameter + !spread:!tagged:handler")
	serverPrefixed, err := container.GetServerPrefixed()
	if err != nil {
		panic(err)
	}
	serverPrefixed.ShowHandlers()
	serverPrefixed.HandleRequest("/orders")

	fmt.Println("\n=== Example Complete ===")
}
