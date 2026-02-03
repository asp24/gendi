package main

import "fmt"

func main() {
	fmt.Println(NewContainer(nil).MustProduct().Name())
}
