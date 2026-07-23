package main

import "fmt"

func main() {
	c := NewContainer(nil)
	// Channel() == "worker" proves SLogPass rewired @logger -> @logger.With("channel","worker").
	fmt.Println(c.MustWorker().Channel())
	// MustLogger compiling/existing proves ExposeAllPass promoted the non-public "logger" service.
	_ = c.MustLogger()
}
