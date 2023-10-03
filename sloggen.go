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
		path    string
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

func (c *config) prepare() {
	if len(c.Levels) > 0 {
		c.Imports = append(c.Imports, "log/slog", "fmt", "strings")
	}
	if len(c.Attrs) > 0 {
		c.Imports = append(c.Imports, "log/slog")
	}
	if c.Logger != nil {
		c.Imports = append(c.Imports, "log/slog", "context", "runtime")
		c.Logger.Levels = c.Levels
		if len(c.Levels) == 0 {
			c.Logger.Levels = map[int]string{
				int(slog.LevelDebug): "debug",
				int(slog.LevelInfo):  "info",
				int(slog.LevelWarn):  "warn",
				int(slog.LevelError): "error",
			}
		}
	}
	slices.Sort(c.Consts)
	slices.Sort(c.Imports)
	c.Imports = slices.Compact(c.Imports)
}

func readFlags(args []string) (*config, error) {
	cfg := config{
		Levels: make(map[int]string),
		Attrs:  make(map[string]string),
	}

	fs := flag.NewFlagSet("sloggen", flag.ContinueOnError)
	fs.StringVar(&cfg.path, "config", "", "read config from the file instead of flags")
	fs.StringVar(&cfg.Pkg, "pkg", "slogx", "the name for the generated package")

	fs.Func("i", "add import", func(s string) error {
		cfg.Imports = append(cfg.Imports, s)
		return nil
	})
	fs.Func("l", "add level (name:severity)", func(s string) error {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return fmt.Errorf("sloggen: -l=%s: invalid value", s)
		}
		severity, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("parsing severity: %w", err)
		}
		cfg.Levels[severity] = parts[0]
		return nil
	})
	fs.Func("c", "add constant", func(s string) error {
		cfg.Consts = append(cfg.Consts, s)
		return nil
	})
	fs.Func("a", "add attribute (key:type)", func(s string) error {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return fmt.Errorf("sloggen: -a=%s: invalid value", s)
		}
		cfg.Attrs[parts[0]] = parts[1]
		return nil
	})
	fs.BoolFunc("logger", "generate a custom Logger type (default false)", func(string) error {
		cfg.Logger = new(logger)
		return nil
	})
	fs.Func("api", `the API style for the Logger's methods ("any" | "attr") (default "any")`, func(s string) error {
		if s != "any" && s != "attr" {
			return fmt.Errorf("sloggen: -api=%s: invalid value", s)
		}
		if cfg.Logger != nil {
			cfg.Logger.AttrAPI = s == "attr"
		}
		return nil
	})
	fs.BoolFunc("ctx", "add context.Context to the Logger's methods (default false)", func(string) error {
		if cfg.Logger != nil {
			cfg.Logger.Context = true
		}
		return nil
	})

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	cfg.prepare()
	return &cfg, nil
}

func readConfig(r io.Reader) (*config, error) {
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

	cfg := config{
		Pkg:     data.Pkg,
		Imports: data.Imports,
		Levels:  make(map[int]string, len(data.Levels)),
		Consts:  data.Consts,
		Attrs:   make(map[string]string, len(data.Attrs)),
	}

	for _, m := range data.Levels {
		name, severity := firstKV(m)
		cfg.Levels[severity] = name
	}
	for _, m := range data.Attrs {
		key, typ := firstKV(m)
		cfg.Attrs[key] = typ
	}
	if data.Logger != nil {
		if data.Logger.API != "any" && data.Logger.API != "attr" {
			return nil, fmt.Errorf("sloggen: logger.api=%s: invalid value", data.Logger.API)
		}
		cfg.Logger = &logger{
			AttrAPI: data.Logger.API == "attr",
			Context: data.Logger.Ctx,
		}
	}

	cfg.prepare()
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
func firstKV[V any](m map[string]V) (string, V) {
	for k, v := range m {
		return k, v
	}
	return "", *new(V)
}
