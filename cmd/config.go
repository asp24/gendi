package cmd

import (
	"flag"
	"fmt"

	"github.com/asp24/gendi/generator"
)

// Config holds CLI configuration
type Config struct {
	ConfigPath       string
	GeneratorOptions generator.Options
}

func (c *Config) RegisterFlags(flags *flag.FlagSet) {
	flags.StringVar(&c.ConfigPath, "config", "", "Root YAML configuration file")
	flags.StringVar(&c.GeneratorOptions.Out, "out", "", "Output directory or file")
	flags.StringVar(&c.GeneratorOptions.Package, "pkg", "", "Go package name")
	flags.StringVar(&c.GeneratorOptions.Container, "container", "Container", "Container struct name")
	flags.BoolVar(&c.GeneratorOptions.Strict, "strict", true, "Enable strict validation")
	flags.StringVar(&c.GeneratorOptions.BuildTags, "build-tags", "", "Go build tags")
	flags.BoolVar(&c.GeneratorOptions.Verbose, "verbose", false, "Verbose logging")
}

// Finalize validates and finalizes the configuration
func (c *Config) Finalize() error {
	if c.ConfigPath == "" {
		return fmt.Errorf("config is required")
	}

	return c.GeneratorOptions.Finalize()
}
