package main

import (
	"errors"
	"flag"

	"github.com/asp24/gendi/generator"
)

type GendiConfig struct {
	ConfigPath       string
	GeneratorOptions generator.Options
}

func (c *GendiConfig) Finalize() error {
	if c.ConfigPath == "" {
		return errors.New("config is required")
	}

	return c.GeneratorOptions.Finalize()
}

func BindFlagsToConfig(flags *flag.FlagSet, gConfig *GendiConfig) error {
	flags.StringVar(&gConfig.ConfigPath, "config", "", "Root YAML configuration file")
	//
	flags.StringVar(&gConfig.GeneratorOptions.Out, "out", "", "Output directory or file")
	flags.StringVar(&gConfig.GeneratorOptions.Package, "pkg", "", "Go package name")
	flags.StringVar(&gConfig.GeneratorOptions.Container, "container", "Container", "Container struct name")
	flags.BoolVar(&gConfig.GeneratorOptions.Strict, "strict", true, "Enable strict validation")
	flags.StringVar(&gConfig.GeneratorOptions.BuildTags, "build-tags", "", "Go build tags")
	flags.BoolVar(&gConfig.GeneratorOptions.Verbose, "verbose", false, "Verbose logging")

	return nil
}
