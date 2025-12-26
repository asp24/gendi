package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	gendi "github.com/asp24/gendi"
	"github.com/asp24/gendi/cli"
	"github.com/asp24/gendi/examples/custom-pass/internal/di"
)

func main() {
	// Register custom compiler passes
	passes := []gendi.Pass{
		&di.AutoTagPass{},
		&di.SLogPass{},
	}

	// Run gendi with custom passes
	if err := cli.Run(flag.CommandLine, passes); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	log.Println("Container generated successfully with custom passes!")
}
