package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/asp24/gendi/cli"
)

func main() {
	if err := cli.Run(flag.CommandLine, nil); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
