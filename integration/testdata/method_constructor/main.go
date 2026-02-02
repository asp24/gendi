package main

import "fmt"

func main() {
	container := NewContainer(nil)
	product, err := container.GetProduct()
	if err != nil {
		panic(err)
	}
	fmt.Println(product.Name())
}
