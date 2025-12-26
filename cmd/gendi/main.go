package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/generator"
	"github.com/asp24/gendi/yaml"
)

func WriteTargetFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func DoAllJob(flags *flag.FlagSet, diPasses ...di.Pass) error {
	var gCfg GendiConfig
	if err := BindFlagsToConfig(flag.CommandLine, &gCfg); err != nil {
		return err
	}

	if err := flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if err := gCfg.Finalize(); err != nil {
		return fmt.Errorf("config finalize: %w", err)
	}

	cfg, err := yaml.LoadConfig(gCfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Apply compiler passes (none in CLI, but users can add via programmatic API)
	cfg, err = di.ApplyPasses(cfg, diPasses)
	if err != nil {
		return fmt.Errorf("apply passes: %w", err)
	}

	gen := generator.New(cfg, gCfg.GeneratorOptions)
	code, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	// After Finalize(), gCfg.GeneratorOptions.Out is the full output file path
	if err := WriteTargetFile(gCfg.GeneratorOptions.Out, code); err != nil {
		return fmt.Errorf("write target file: %w", err)
	}

	if gCfg.GeneratorOptions.Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "generated %s\n", gCfg.GeneratorOptions.Out)
	}

	return nil
}

func exitf(msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func main() {
	if err := DoAllJob(flag.CommandLine); err != nil {
		exitf("%v", err)
	}
}
