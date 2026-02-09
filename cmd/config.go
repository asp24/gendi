package cmd

import (
	"flag"
	"fmt"

	"github.com/asp24/gendi/pipeline"
)

// Config holds CLI configuration
type Config struct {
	ConfigPath string
	Options    pipeline.Options
}

func (c *Config) RegisterFlags(flags *flag.FlagSet) {
	flags.StringVar(&c.ConfigPath, "config", "", "Root YAML configuration file")
	flags.StringVar(&c.Options.Out, "out", "", "Output directory or file")
	flags.StringVar(&c.Options.Package, "pkg", "", "Go package name")
	flags.StringVar(&c.Options.Container, "container", "Container", "Container struct name")
	flags.BoolVar(&c.Options.Strict, "strict", true, "Enable strict validation")
	flags.StringVar(&c.Options.BuildTags, "build-tags", "", "Go build tags")
	flags.BoolVar(&c.Options.Verbose, "verbose", false, "Verbose logging")
}

// Finalize validates and finalizes the configuration
func (c *Config) Finalize() error {
	if c.ConfigPath == "" {
		return fmt.Errorf("config is required")
	}

	return c.Options.Finalize()
}
