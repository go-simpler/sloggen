package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go-simpler.org/errorsx"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() (err error) {
	var cfgPath string

	fs := flag.NewFlagSet("slog-gen", flag.ContinueOnError)
	fs.StringVar(&cfgPath, "config", ".slog.yml", "path to config")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parsing flags: %w", err)
	}

	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer errorsx.Close(cfgFile, &err)

	cfg, err := readConfig(cfgFile)
	if err != nil {
		return err
	}

	if err := os.Mkdir(cfg.Pkg, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("mkdir: %w", err)
	}

	genFile, err := os.Create(filepath.Join(cfg.Pkg, cfg.Pkg+".go"))
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer errorsx.Close(genFile, &err)

	return writeCode(genFile, cfg)
}
