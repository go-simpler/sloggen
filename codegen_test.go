package main

import (
	"bytes"
	"strings"
	"testing"

	"go-simpler.org/assert"
	. "go-simpler.org/assert/dotimport"
)

//go:generate go run main.go codegen.go --config=.slog.example.yml

func Test_readConfig(t *testing.T) {
	r := strings.NewReader(`
pkg: test
imports: [time]
levels: {custom: -8}
consts: [foo]
attrs:
  bar: time.Time
  baz: time.Duration
`)

	want := &config{
		Pkg:     "test",
		Imports: []string{"log/slog", "time"},
		Levels:  []level{{Name: "custom", Severity: -8}},
		Consts:  []string{"foo"},
		Attrs: []attr{
			{Key: "bar", Type: "time.Time"},
			{Key: "baz", Type: "time.Duration"},
		},
	}

	got, err := readConfig(r)
	assert.NoErr[F](t, err)
	assert.Equal[E](t, got, want)
}

func Test_writeCode(t *testing.T) {
	cfg := config{
		Pkg:     "test",
		Imports: []string{"log/slog"},
		Levels:  []level{{Name: "custom", Severity: -8}},
		Consts:  []string{"foo"},
		Attrs: []attr{
			{Key: "bar", Type: "int"},
			{Key: "baz", Type: "error"},
		},
	}

	const src = `// Code generated by go-simpler.org/slog-gen. DO NOT EDIT.

package test

import "log/slog"

const LevelCustom = slog.Level(-8)

const Foo = "foo"

func Bar(value int) slog.Attr {
	return slog.Int("bar", value)
}

func Baz(value error) slog.Attr {
	return slog.Any("baz", value)
}
`

	var buf bytes.Buffer
	err := writeCode(&buf, &cfg)
	assert.NoErr[F](t, err)
	assert.Equal[E](t, buf.String(), src)
}
