package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asp24/go-sf-di"
	"github.com/asp24/go-sf-di/internal/generator"
)

func main() {
	var (
		configPath  = flag.String("config", "", "Root YAML configuration file")
		outPath     = flag.String("out", "", "Output directory or file")
		pkgName     = flag.String("pkg", "", "Go package name")
		container   = flag.String("container", "Container", "Container struct name")
		strict      = flag.Bool("strict", true, "Enable strict validation")
		buildTags   = flag.String("build-tags", "", "Go build tags")
		verbose     = flag.Bool("verbose", false, "Verbose logging")
	)
	flag.Parse()

	if *configPath == "" || *outPath == "" || *pkgName == "" {
		exitf("--config, --out, and --pkg are required")
	}

	cfg, err := di.LoadConfig(*configPath)
	if err != nil {
		exitf("load config: %v", err)
	}

	outFile, err := outputFile(*outPath)
	if err != nil {
		exitf("output path: %v", err)
	}

	gen := generator.New(cfg, generator.Options{
		Out:       *outPath,
		Package:   *pkgName,
		Container: *container,
		Strict:    *strict,
		BuildTags: *buildTags,
		Verbose:   *verbose,
	}, nil)

	code, err := gen.Generate()
	if err != nil {
		exitf("generate: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		exitf("create output dir: %v", err)
	}
	if err := os.WriteFile(outFile, code, 0o644); err != nil {
		exitf("write output: %v", err)
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "generated %s\n", outFile)
	}
}

func outputFile(out string) (string, error) {
	if strings.HasSuffix(out, ".go") {
		return out, nil
	}
	info, err := os.Stat(out)
	if err == nil && info.IsDir() {
		return filepath.Join(out, "container_gen.go"), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	// Treat as directory when it doesn't exist and doesn't end with .go.
	return filepath.Join(out, "container_gen.go"), nil
}

func exitf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
