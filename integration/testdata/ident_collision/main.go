package main

import "fmt"

func main() {
	container := NewContainer(nil)
	consumer, err := container.GetConsumer()
	if err != nil {
		panic(err)
	}
	fmt.Println(consumer.Info())
}
