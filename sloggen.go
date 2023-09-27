package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"slices"
	"strconv"
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

func readConfig(r io.Reader, args []string) (*config, error) {
	var data struct {
		Pkg     string              `yaml:"pkg"`
		Imports []string            `yaml:"imports"`
		Levels  []map[string]int    `yaml:"levels"`
		Consts  []string            `yaml:"consts"`
		Attrs   []map[string]string `yaml:"attrs"`
		Logger  *struct {
			API string `yaml:"api"`
			Ctx bool   `yaml:"ctx"`
		} `yaml:"logger"`
	}
	if err := yaml.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	fs := flag.NewFlagSet("sloggen", flag.ContinueOnError)
	fs.StringVar(&data.Pkg, "pkg", "slogx", "set package name")
	fs.Func("i", "add import", func(s string) error {
		data.Imports = append(data.Imports, s)
		return nil
	})
	fs.Func("l", "add level (name:severity)", func(s string) error {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return fmt.Errorf("sloggen: -l=%s: invalid value", s)
		}
		i, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("parsing severity: %w", err)
		}
		data.Levels = append(data.Levels, map[string]int{parts[0]: i})
		return nil
	})
	fs.Func("c", "add constant", func(s string) error {
		data.Consts = append(data.Consts, s)
		return nil
	})
	fs.Func("a", "add attribute (key:type)", func(s string) error {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return fmt.Errorf("sloggen: -a=%s: invalid value", s)
		}
		data.Attrs = append(data.Attrs, map[string]string{parts[0]: parts[1]})
		return nil
	})
	fs.BoolFunc("logger", "add Logger type (default false)", func(string) error {
		data.Logger = new(struct {
			API string `yaml:"api"`
			Ctx bool   `yaml:"ctx"`
		})
		return nil
	})
	fs.Func("api", `set API style for Logger methods ("any" | "attr") (default "any")`, func(s string) error {
		if data.Logger != nil {
			data.Logger.API = s
		}
		return nil
	})
	fs.BoolFunc("ctx", "add context.Context to Logger methods (default false)", func(string) error {
		if data.Logger != nil {
			data.Logger.Ctx = true
		}
		return nil
	})
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	cfg := config{
		Pkg:     data.Pkg,
		Imports: data.Imports,
		Levels:  make(map[int]string, len(data.Levels)),
		Consts:  data.Consts,
		Attrs:   make(map[string]string, len(data.Attrs)),
		Logger:  nil,
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
			Context: data.Logger.Ctx,
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
