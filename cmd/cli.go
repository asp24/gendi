package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/pipeline"
	"github.com/asp24/gendi/srcloc"
	"github.com/asp24/gendi/yaml"
)

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

func Generate(cfg Config, passes []di.Pass) error {
	if err := cfg.Finalize(); err != nil {
		return fmt.Errorf("config finalize: %w", err)
	}

	diCfg, err := yaml.LoadConfig(cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	diCfg, err = di.ApplyPasses(diCfg, passes)
	if err != nil {
		return fmt.Errorf("apply passes: %w", err)
	}

	code, err := pipeline.Emit(diCfg, cfg.Options)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	// After Finalize(), cfg.Options.Out is the full output file path
	if err := WriteTargetFile(cfg.Options.Out, code); err != nil {
		return fmt.Errorf("write target file: %w", err)
	}

	if cfg.Options.Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "generated %s\n", cfg.Options.Out)
	}

	return nil
}

// Run executes the full gendi workflow with optional compiler passes
func Run(flags *flag.FlagSet, passes []di.Pass) error {
	var cfg Config
	cfg.RegisterFlags(flags)

	if err := flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	return Generate(cfg, passes)
}

func MustRun(flags *flag.FlagSet, passes []di.Pass) {
	err := Run(flags, passes)
	if err == nil {
		return
	}

	errRenderer := srcloc.NewRenderer()
	_, _ = os.Stderr.WriteString(errRenderer.RenderError(err, 4))

	os.Exit(1)
}
