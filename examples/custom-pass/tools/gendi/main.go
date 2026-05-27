package main

import (
	"flag"

	gendi "github.com/asp24/gendi"
	"github.com/asp24/gendi/cmd"
	"github.com/asp24/gendi/examples/custom-pass/internal/di"
	"github.com/asp24/gendi/stdlib"
)

func main() {
	// Register custom compiler passes
	passes := []gendi.Pass{
		&di.AutoTagPass{},
		&stdlib.SLogPass{},
	}

	cmd.MustRun(flag.CommandLine, passes, nil)
}
