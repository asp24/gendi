package stdlib

import (
	"io"
	"os"
)

// NewStdout returns os.Stdout as an io.Writer.
// Useful for configuring loggers to write to standard output.
//
// Example:
//
//	services:
//	  stdlib.stdout:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewStdout"
//	  	shared: false
func NewStdout() io.Writer {
	return os.Stdout
}

// NewStderr returns os.Stderr as an io.Writer.
// Useful for configuring loggers to write to standard error.
//
// Example:
//
//	services:
//	  stdlib.stderr:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewStderr"
//	  	shared: false
func NewStderr() io.Writer {
	return os.Stderr
}
