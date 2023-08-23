package main

import (
	"io"
	"log/slog"
	"time"

	"go-simpler.org/slog-gen/slogattr"
)

//go:generate go run main.go -config=.slog.yml

func Example() {
	slog.Info("example",
		slogattr.UserId(42),
		slogattr.CreatedAt(time.Now()),
		slogattr.Err(io.EOF))
}
