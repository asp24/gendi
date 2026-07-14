package cmd

import (
	"flag"
	"fmt"
	"sort"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/pipeline"
)

// Config holds CLI configuration
type Config struct {
	ConfigPath    string
	Options       pipeline.Options
	EnabledPasses map[string]struct{}
}

func (c *Config) RegisterFlags(flags *flag.FlagSet) {
	flags.StringVar(&c.ConfigPath, "config", "", "Root YAML configuration file")
	flags.StringVar(&c.Options.Out, "out", "", "Output directory or file")
	flags.StringVar(&c.Options.Package, "pkg", "", "Go package name")
	flags.StringVar(&c.Options.Container, "container", "Container", "Container struct name")
	flags.StringVar(&c.Options.BuildTags, "build-tags", "", "Go build tags")
	flags.BoolVar(&c.Options.Verbose, "verbose", false, "Verbose logging")
	flags.Var(&stringSetFlag{values: &c.EnabledPasses}, "enable-pass", "Enable a specific compiler pass (can be specified multiple times)")
}

func (c *Config) resolvePasses(passes, selectablePasses []di.Pass) ([]di.Pass, error) {
	known := make(map[string]struct{}, len(selectablePasses))
	for _, p := range selectablePasses {
		known[p.Name()] = struct{}{}
	}

	var unknown []string
	for name := range c.EnabledPasses {
		if _, ok := known[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("--enable-pass: unknown pass %q", unknown[0])
	}

	result := make([]di.Pass, 0, len(passes)+len(selectablePasses))
	included := make(map[string]struct{}, len(passes)+len(selectablePasses))
	for _, p := range passes {
		name := p.Name()
		if _, ok := included[name]; ok {
			continue
		}

		result = append(result, p)
		included[name] = struct{}{}
	}

	for _, p := range selectablePasses {
		name := p.Name()
		if _, ok := included[name]; ok {
			continue
		}

		_, enabled := c.EnabledPasses[name]
		if !enabled {
			continue
		}

		result = append(result, p)
		included[name] = struct{}{}
	}

	return result, nil
}

// Finalize validates and finalizes the configuration
func (c *Config) Finalize() error {
	if c.ConfigPath == "" {
		return fmt.Errorf("config is required")
	}

	return c.Options.Finalize()
}
