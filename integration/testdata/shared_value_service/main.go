package main

import "fmt"

func main() {
	container := NewContainer(nil)

	c1, err := container.GetCounter()
	if err != nil {
		panic(err)
	}

	c2, err := container.GetCounter()
	if err != nil {
		panic(err)
	}

	// Both calls return the cached value (same ID) and the constructor ran once,
	// proving the shared value getter's Init-flag caching.
	fmt.Printf("id1=%d id2=%d builds=%d\n", c1.ID, c2.ID, BuildCount())
}
