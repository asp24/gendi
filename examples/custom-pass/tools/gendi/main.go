package main

import (
	"flag"

	gendi "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/cmd"
	"github.com/gendi-org/gendi/examples/custom-pass/internal/di"
	"github.com/gendi-org/gendi/stdlib"
)

func main() {
	// Register custom compiler passes
	passes := []gendi.Pass{
		&di.AutoTagPass{},
		&stdlib.SLogPass{},
	}

	cmd.MustRun(flag.CommandLine, passes, nil)
}
