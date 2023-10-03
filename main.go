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
	cfg, err := readFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if err := os.Mkdir(cfg.Pkg, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	f, err := os.Create(filepath.Join(cfg.Pkg, cfg.Pkg+".go"))
	if err != nil {
		return err
	}
	defer errorsx.Close(f, &err)

	return writeCode(f, cfg)
}
