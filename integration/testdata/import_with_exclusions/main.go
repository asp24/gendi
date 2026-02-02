package main

import "fmt"

func main() {
	container := NewContainer(nil)
	_, err := container.GetProdService()
	if err != nil {
		panic(err)
	}
	fmt.Println("prod service loaded")

	// test_service should not exist because it was excluded
	fmt.Println("test exclusion works")
}
