package main

import "fmt"

func main() {
	fmt.Printf("service banner is: %s\n", NewContainer(nil).MustService().GetBanner())
}
