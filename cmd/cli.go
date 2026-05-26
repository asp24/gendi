package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/pipeline"
	"github.com/asp24/gendi/srcloc"
	"github.com/asp24/gendi/stdlib"
	"github.com/asp24/gendi/yaml"
)

func BuiltinSelectablePasses() []di.SelectablePass {
	return []di.SelectablePass{
		stdlib.NewSLogPass(false),
		&di.ExposeAllPass{},
	}
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

func Generate(cfg Config, passes []di.Pass) error {
	if err := cfg.Finalize(); err != nil {
		return fmt.Errorf("config finalize: %w", err)
	}

	diCfg, err := yaml.LoadConfig(cfg.ConfigPath)
	if err != nil {
		return srcloc.AddContext(err, "load config")
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
func Run(flags *flag.FlagSet, selectablePasses []di.SelectablePass) error {
	var cfg Config
	cfg.RegisterFlags(flags)

	if err := flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	passes, err := cfg.Passes.resolvePasses(selectablePasses)
	if err != nil {
		return fmt.Errorf("resolve passes: %w", err)
	}

	return Generate(cfg, passes)
}

func PrintErrorAndExit(err error) {
	if err == nil {
		os.Exit(0)
	}

	errRenderer := srcloc.NewRenderer()
	_, _ = os.Stderr.WriteString(errRenderer.RenderError(err, 4))

	os.Exit(1)
}

func MustRun(flags *flag.FlagSet, selectablePasses []di.SelectablePass) {
	PrintErrorAndExit(Run(flags, selectablePasses))
}
