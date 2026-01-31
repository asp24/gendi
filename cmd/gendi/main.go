package main

import (
	"flag"

	"github.com/asp24/gendi/cmd"
)

func main() {
	cmd.MustRun(flag.CommandLine, nil)
}
