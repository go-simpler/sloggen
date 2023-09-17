package main

import (
	"bytes"
	"cmp"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"slices"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var tmpl = template.Must(template.New("").Funcs(funcs).Parse(
	`// Code generated by go-simpler.org/slog-gen. DO NOT EDIT.

package {{.Pkg}}

{{range .Imports -}}
import "{{.}}"
{{end}}

{{range .Levels -}}
const Level{{snakeToCamel .Name}} = slog.Level({{.Severity}})
{{end}}

{{range .Consts -}}
const {{snakeToCamel .}} = "{{.}}"
{{end}}

{{range .Attrs}}
func {{snakeToCamel .Key}}(value {{.Type}}) slog.Attr {
	return slog.{{slogFunc .Type}}("{{.Key}}", value)
}
{{end}}

{{if .HasCustomLevels}}
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToUpper(s) {
	{{range .Levels -}}
	case "{{toUpper .Name}}":
		return Level{{snakeToCamel .Name}}, nil
	{{end -}}
	default:
		return 0, fmt.Errorf("slog: level string %q: unknown name", s)
	}
}

func ReplaceAttr(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key != slog.LevelKey {
		return attr
	}
	switch attr.Value.Any().(slog.Level) {
	{{range .Levels -}}
	case Level{{snakeToCamel .Name}}:
		attr.Value = slog.StringValue("{{toUpper .Name}}")
	{{end -}}
	}
	return attr
}
{{end}}`,
))

//nolint:staticcheck // SA1019: strings.Title is deprecated but works just fine here.
var funcs = template.FuncMap{
	"toUpper": strings.ToUpper,
	"snakeToCamel": func(s string) string {
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

type (
	config struct {
		Pkg             string
		Imports         []string
		Levels          []level
		Consts          []string
		Attrs           []attr
		HasCustomLevels bool
	}
	level struct {
		Name     string
		Severity int
	}
	attr struct {
		Key  string
		Type string
	}
)

func readConfig(r io.Reader) (*config, error) {
	cfg := struct {
		Pkg     string            `yaml:"pkg"`
		Imports []string          `yaml:"imports"`
		Levels  map[string]int    `yaml:"levels"` // name:severity
		Consts  []string          `yaml:"consts"`
		Attrs   map[string]string `yaml:"attrs"` // key:type
	}{
		Pkg: "log",
	}
	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	hasCustomLevels := false
	slogLevels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}

	levels := make([]level, 0, len(cfg.Levels))
	for name, severity := range cfg.Levels {
		levels = append(levels, level{Name: name, Severity: severity})
		if !slices.Contains(slogLevels, slog.Level(severity)) {
			hasCustomLevels = true
		}
	}

	attrs := make([]attr, 0, len(cfg.Attrs))
	for key, typ := range cfg.Attrs {
		attrs = append(attrs, attr{Key: key, Type: typ})
	}

	if len(attrs) > 0 || len(levels) > 0 {
		cfg.Imports = append(cfg.Imports, "log/slog")
	}
	if hasCustomLevels {
		cfg.Imports = append(cfg.Imports, "fmt", "strings") // for ParseLevel().
	}

	slices.Sort(cfg.Imports)
	slices.Sort(cfg.Consts)
	slices.SortFunc(levels, func(l1, l2 level) int {
		return cmp.Compare(l1.Severity, l2.Severity)
	})
	slices.SortFunc(attrs, func(a1, a2 attr) int {
		return cmp.Compare(a1.Key, a2.Key)
	})

	return &config{
		Pkg:             cfg.Pkg,
		Imports:         cfg.Imports,
		Levels:          levels,
		Consts:          cfg.Consts,
		Attrs:           attrs,
		HasCustomLevels: hasCustomLevels,
	}, nil
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
