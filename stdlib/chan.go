package stdlib

// NewChan creates a buffered channel of type T with the specified buffer size.
// Use size=0 for an unbuffered channel.
//
// Example:
//
//	services:
//	  events:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewChan[github.com/myapp.Event]"
//	      args: [100]
func NewChan[T any](size int) chan T {
	return make(chan T, size)
}
