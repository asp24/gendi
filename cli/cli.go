package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/generator"
	"github.com/asp24/gendi/yaml"
)

// Config holds CLI configuration
type Config struct {
	ConfigPath       string
	GeneratorOptions generator.Options
}

// Finalize validates and finalizes the configuration
func (c *Config) Finalize() error {
	if c.ConfigPath == "" {
		return fmt.Errorf("config is required")
	}

	return c.GeneratorOptions.Finalize()
}

// BindFlags binds command-line flags to the config
func BindFlags(flags *flag.FlagSet, cfg *Config) {
	flags.StringVar(&cfg.ConfigPath, "config", "", "Root YAML configuration file")
	flags.StringVar(&cfg.GeneratorOptions.Out, "out", "", "Output directory or file")
	flags.StringVar(&cfg.GeneratorOptions.Package, "pkg", "", "Go package name")
	flags.StringVar(&cfg.GeneratorOptions.Container, "container", "Container", "Container struct name")
	flags.BoolVar(&cfg.GeneratorOptions.Strict, "strict", true, "Enable strict validation")
	flags.StringVar(&cfg.GeneratorOptions.BuildTags, "build-tags", "", "Go build tags")
	flags.BoolVar(&cfg.GeneratorOptions.Verbose, "verbose", false, "Verbose logging")
}

// WriteTargetFile writes data to the specified file path
func WriteTargetFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Run executes the full gendi workflow with optional compiler passes
func Run(flags *flag.FlagSet, passes []di.Pass) error {
	var cfg Config
	BindFlags(flags, &cfg)

	if err := flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if err := cfg.Finalize(); err != nil {
		return fmt.Errorf("config finalize: %w", err)
	}

	diCfg, err := yaml.LoadConfig(cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Apply compiler passes
	diCfg, err = di.ApplyPasses(diCfg, passes)
	if err != nil {
		return fmt.Errorf("apply passes: %w", err)
	}

	gen := generator.New(diCfg, cfg.GeneratorOptions)
	code, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	// After Finalize(), cfg.GeneratorOptions.Out is the full output file path
	if err := WriteTargetFile(cfg.GeneratorOptions.Out, code); err != nil {
		return fmt.Errorf("write target file: %w", err)
	}

	if cfg.GeneratorOptions.Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "generated %s\n", cfg.GeneratorOptions.Out)
	}

	return nil
}
