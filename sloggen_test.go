package main

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"go-simpler.org/assert"
	. "go-simpler.org/assert/dotimport"
	"go-simpler.org/sloggen/example"
)

//go:generate go run main.go sloggen.go --config=.slog.example.yml

var cfg = config{
	Pkg:     "test",
	Imports: []string{"fmt", "log/slog", "strings", "time"},
	Levels:  map[int]string{-8: "custom"},
	Consts:  []string{"foo"},
	Attrs: map[string]string{
		"bar": "time.Time",
		"baz": "time.Duration",
	},
}

func Test_readConfig(t *testing.T) {
	r := strings.NewReader(`
pkg: test
imports:
  - time
levels:
  - custom: -8
consts:
  - foo
attrs:
  - bar: time.Time
  - baz: time.Duration
`)

	got, err := readConfig(r)
	assert.NoErr[F](t, err)
	assert.Equal[E](t, got, &cfg)
}

func Test_writeCode(t *testing.T) {
	const src = `// Code generated by go-simpler.org/sloggen. DO NOT EDIT.

package test

import "fmt"
import "log/slog"
import "strings"
import "time"

const LevelCustom = slog.Level(-8)

const Foo = "foo"

func Bar(value time.Time) slog.Attr     { return slog.Time("bar", value) }
func Baz(value time.Duration) slog.Attr { return slog.Duration("baz", value) }

func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToUpper(s) {
	case "CUSTOM":
		return LevelCustom, nil
	default:
		return 0, fmt.Errorf("slog: level string %q: unknown name", s)
	}
}

func ReplaceAttr(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key != slog.LevelKey {
		return attr
	}
	switch attr.Value.Any().(slog.Level) {
	case LevelCustom:
		attr.Value = slog.StringValue("CUSTOM")
	}
	return attr
}
`
	var buf bytes.Buffer
	err := writeCode(&buf, &cfg)
	assert.NoErr[F](t, err)
	assert.Equal[E](t, buf.String(), src)
}

func TestParseLevel(t *testing.T) {
	level, err := example.ParseLevel("TRACE")
	assert.NoErr[F](t, err)
	assert.Equal[E](t, level, example.LevelTrace)
}

func TestReplaceAttr(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: example.LevelTrace,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return example.ReplaceAttr(groups, attr)
		},
	})

	logger := slog.New(handler)
	logger.Log(context.Background(), example.LevelTrace, "test")
	assert.Equal[E](t, buf.String(), "level=TRACE msg=test\n")
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		AddSource: true,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			if attr.Key == slog.SourceKey {
				src := attr.Value.Any().(*slog.Source)
				src.File = filepath.Base(src.File)
			}
			return attr
		},
	})

	logger := example.Logger{Logger: slog.New(handler)}
	logger.Info(context.Background(), "test")
	assert.Equal[E](t, buf.String(), "level=INFO source=sloggen_test.go:131 msg=test\n")
}
