package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/asp24/gendi/cmd"
)

func main() {
	if err := cmd.Run(flag.CommandLine, nil); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
