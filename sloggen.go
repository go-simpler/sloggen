package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"slices"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed template.go.tmpl
	src  string
	tmpl = template.Must(template.New("").Funcs(funcs).Parse(src))

	//nolint:staticcheck // SA1019: strings.Title is deprecated but works just fine here.
	funcs = template.FuncMap{
		"title": strings.Title,
		"upper": strings.ToUpper,
		"camel": func(s string) string {
			parts := strings.Split(s, "_")
			for i := range parts {
				parts[i] = strings.Title(parts[i])
			}
			return strings.Join(parts, "")
		},
		"slogFunc": func(typ string) string {
			switch s := strings.Title(strings.TrimPrefix(typ, "time.")); s {
			case "String", "Int64", "Int", "Uint64", "Float64", "Bool", "Time", "Duration":
				return s
			default:
				return "Any"
			}
		},
	}
)

// NOTE: when iterating over a map, text/template visits the elements in sorted key order.
type (
	config struct {
		Pkg     string
		Imports []string
		Levels  map[int]string // severity:name
		Consts  []string
		Attrs   map[string]string // key:type
		Logger  *logger
	}
	logger struct {
		Levels  map[int]string
		AttrAPI bool
		Context bool
	}
)

func readConfig(r io.Reader) (*config, error) {
	var data struct {
		Pkg     string              `yaml:"pkg"`
		Imports []string            `yaml:"imports"`
		Levels  []map[string]int    `yaml:"levels"`
		Consts  []string            `yaml:"consts"`
		Attrs   []map[string]string `yaml:"attrs"`
		Logger  *struct {
			API     string `yaml:"api"`
			Context bool   `yaml:"context"`
		} `yaml:"logger"`
	}
	if err := yaml.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	cfg := config{
		Pkg:     data.Pkg,
		Imports: data.Imports,
		Levels:  make(map[int]string, len(data.Levels)),
		Consts:  data.Consts,
		Attrs:   make(map[string]string, len(data.Attrs)),
		Logger:  nil,
	}
	if cfg.Pkg == "" {
		cfg.Pkg = "slogx"
	}

	for _, m := range data.Levels {
		name, severity := getKV(m)
		cfg.Levels[severity] = name
	}

	for _, m := range data.Attrs {
		key, typ := getKV(m)
		cfg.Attrs[key] = typ
	}

	if data.Logger != nil {
		cfg.Imports = append(cfg.Imports, "context", "runtime")
		cfg.Logger = &logger{
			Levels:  cfg.Levels,
			AttrAPI: false,
			Context: data.Logger.Context,
		}
		if len(cfg.Levels) == 0 {
			cfg.Logger.Levels = map[int]string{
				int(slog.LevelDebug): "debug",
				int(slog.LevelInfo):  "info",
				int(slog.LevelWarn):  "warn",
				int(slog.LevelError): "error",
			}
		}
		switch data.Logger.API {
		case "any":
		case "attr":
			cfg.Logger.AttrAPI = true
		default:
			return nil, fmt.Errorf("sloggen: %q: invalid logger.api value", data.Logger.API)
		}
	}

	if len(cfg.Attrs) > 0 || len(cfg.Levels) > 0 || cfg.Logger != nil {
		cfg.Imports = append(cfg.Imports, "log/slog")
	}
	if len(cfg.Levels) > 0 {
		cfg.Imports = append(cfg.Imports, "fmt", "strings")
	}

	slices.Sort(cfg.Consts)
	slices.Sort(cfg.Imports)
	cfg.Imports = slices.Compact(cfg.Imports)

	return &cfg, nil
}

func writeCode(w io.Writer, cfg *config) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting code: %w", err)
	}

	if _, err := w.Write(src); err != nil {
		return fmt.Errorf("writing code: %w", err)
	}

	return nil
}

//nolint:gocritic // unnamedResult: generics false positive.
func getKV[V any](m map[string]V) (string, V) {
	for k, v := range m {
		return k, v
	}
	return "", *new(V)
}
