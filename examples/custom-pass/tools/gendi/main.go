package main

import (
	"flag"

	gendi "github.com/asp24/gendi"
	"github.com/asp24/gendi/cmd"
	"github.com/asp24/gendi/examples/custom-pass/internal/di"
)

func main() {
	// Register custom compiler passes
	passes := []gendi.Pass{
		&di.AutoTagPass{},
		&di.SLogPass{},
	}

	cmd.MustRun(flag.CommandLine, passes)
}
