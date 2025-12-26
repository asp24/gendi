package main

import (
	"flag"
	"fmt"
	"os"

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

	// Run gendi with custom passes
	if err := cmd.Run(flag.CommandLine, passes); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
