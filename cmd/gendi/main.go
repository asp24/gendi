package main

import (
	"flag"

	"github.com/gendi-org/gendi/cmd"
)

func main() {
	cmd.MustRun(flag.CommandLine, nil, cmd.BuiltinSelectablePasses())
}
