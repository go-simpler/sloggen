# sloggen

[![checks](https://github.com/go-simpler/sloggen/actions/workflows/checks.yml/badge.svg)](https://github.com/go-simpler/sloggen/actions/workflows/checks.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/go-simpler.org/sloggen.svg)](https://pkg.go.dev/go-simpler.org/sloggen)
[![goreportcard](https://goreportcard.com/badge/go-simpler.org/sloggen)](https://goreportcard.com/report/go-simpler.org/sloggen)
[![codecov](https://codecov.io/gh/go-simpler/sloggen/branch/main/graph/badge.svg)](https://codecov.io/gh/go-simpler/sloggen)

## 📌 About

When using `log/slog` in a production-grade project, it is useful to write helpers to avoid human error in the keys.

```go
slog.Info("a user has logged in", "user_id", 42)
slog.Info("a user has logged out", "user_ip", 42) // oops :(
```

Depending on your code style, these can be simple constants (if you prefer key-value arguments)...

```go
const UserId = "user_id"
```

...or constructors for `slog.Attr` (if you're a safety/performance advocate).

```go
func UserId(value int) slog.Attr {
    return slog.Int("user_id", value)
}
```

`sloggen` generates such code for you based on a simple config (a single source of truth),
which makes it easy to share domain-specific helpers between related (micro)services.

## 📦 Install

Create and fill in the `.slog.yml` config based on the example,
then add the following directive to any `.go` file and run `go generate ./...`.

```go
//go:generate go run go-simpler.org/sloggen@latest --config=.slog.yml
```

To get started, see the `.slog.example.yml` file and the `example` directory.
