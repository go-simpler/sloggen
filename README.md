# sloggen

[![checks](https://github.com/go-simpler/sloggen/actions/workflows/checks.yml/badge.svg)](https://github.com/go-simpler/sloggen/actions/workflows/checks.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/go-simpler.org/sloggen.svg)](https://pkg.go.dev/go-simpler.org/sloggen)
[![goreportcard](https://goreportcard.com/badge/go-simpler.org/sloggen)](https://goreportcard.com/report/go-simpler.org/sloggen)
[![codecov](https://codecov.io/gh/go-simpler/sloggen/branch/main/graph/badge.svg)](https://codecov.io/gh/go-simpler/sloggen)

Generate domain-specific helpers for `log/slog`.

## 📌 About

When using `log/slog` in a production-grade project, it is useful to write helpers to prevent typos in the keys:

```go
slog.Info("a user has logged in", "user_id", 42)
slog.Info("a user has logged out", "user_ip", 42) // oops :(
```

Depending on your code style, these can be simple constants (if you prefer key-value pairs)...

```go
const UserId = "user_id"
```

...or custom `slog.Attr` constructors (if you're a safety/performance advocate):

```go
func UserId(value int) slog.Attr { return slog.Int("user_id", value) }
```

`sloggen` generates such helpers for you, so you don't have to write them manually.

---

The default `log/slog` levels cover most use cases, but at some point you may want to introduce custom levels that better suit your app.
At first glance, this is as simple as defining a constant:

```go
const LevelAlert = slog.Level(12)
```

However, custom levels are treated differently than the first-class citizens `Debug`/`Info`/`Warn`/`Error`:

```go
slog.Log(nil, LevelAlert, "msg") // want "ALERT msg"; got "ERROR+4 msg"
```

`sloggen` solves this inconvenience by generating not only the levels themselves, but also the necessary helpers.

Unfortunately, the only way to use such levels is the `Log` method, which is quite verbose.
`sloggen` can generate a custom `Logger` type so that custom levels can be used just like the builtin ones:

```go
// before:
logger.Log(nil, LevelAlert, "msg", "key", "value")
// after:
logger.Alert("msg", "key", "value")
```

Additionally, there are options to choose the API style of the arguments (`...any` or `...slog.Attr`) and to add/remove `context.Context` as the first parameter.
This allows you to adjust the logging API to your own code style without sacrificing convenience.

> 💡 Various API rules for `log/slog` can be enforced by the [`sloglint`][1] linter. Give it a try too!

## 🚀 Features

* Generate key constants and `slog.Attr` constructors
* Generate custom levels with helpers for parsing/printing
* Generate a custom `Logger` type with methods for custom levels
* Codegen-based, so no runtime dependency introduced

## 📦 Install

Add the following directive to any `.go` file and run `go generate ./...`.

```go
//go:generate go run go-simpler.org/sloggen@<version> [flags]
```

Where `<version>` is the version of `sloggen` itself (use `latest` for automatic updates) and `[flags]` is the list of [available options](#help).

## 📋 Usage

There are two ways to provide options to `sloggen`: CLI flags and a `.yml` config file.
The former works best for few options and requires only a single `//go:generate` directive.
For many options it may be more convenient to use a config file, since `go generate` does not support multiline commands.
The config file can also be reused between several (micro)services if they share the same domain.

To get started, see the [`example_test.go`](example_test.go) file and the [`example`](example) directory.

### Key constants

The `-c` flag (or the `consts` field) is used to generate a key constant.
For example, `-c=used_id` results in:

```go
const UserId = "user_id"
```

### Attribute constructors

The `-a` flag (or the `attrs` field) is used to generate a custom `slog.Attr` constructor.
For example, `-a=used_id:int` results in:

```go
func UserId(value int) slog.Attr { return slog.Int("user_id", value) }
```

### Custom levels

The `-l` flag (or the `levels` field) is used to generate a custom `slog.Level`.
For example, `-l=alert:12` results in:

```go
const LevelAlert = slog.Level(12)

func ParseLevel(s string) (slog.Level, error) {...}
func RenameLevels(_ []string, attr slog.Attr) slog.Attr {...}
```

The `ParseLevel` function should be used to parse the level from a string (e.g. from an environment variable):

```go
level, err := slogx.ParseLevel("ALERT")
```

The `RenameLevels` function should be used as `slog.HandlerOptions.ReplaceAttr` to print custom level names correctly:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level:       level,
    ReplaceAttr: slogx.RenameLevels,
}))
```

### Custom Logger

The `-logger` flag (or the `logger` field) is used to generate a custom `Logger` type with methods for custom levels.

The `-api` flag (or the `logger.api` field) is used to choose the API style of the arguments: `any` for `...any` (key-value pairs) and `attr` for `...slog.Attr`.

The `-ctx` flag (or the `logger.ctx` field) is used to add or remove `context.Context` as the first parameter.

For example, `-l=alert:12 -logger -api=attr -ctx` results in:

```go
type Logger struct{ handler slog.Handler }

func New(h slog.Handler) *Logger { return &Logger{handler: h} }

func (l *Logger) Alert(ctx context.Context, msg string, attrs ...slog.Attr) {...}
```

The generated `Logger` has all the utility methods of the original `slog.Logger`, including `Enabled()`, `With()` and `WithGroup()`.

Since `Logger` is just a frontend, you can always fall back to `slog.Logger` (e.g. to pass it to a library) using the `Handler()` method:

```go
slog.New(logger.Handler())
```

### Help

```shell
Usage: sloggen [flags]

Flags:
    -config <path>       read config from the file instead of flags
    -dir <path>          change the working directory before generating files
    -pkg <name>          the name for the generated package (default: slogx)
    -i <import>          add import
    -l <name:severity>   add level
    -c <key>             add constant
    -a <key:type>        add attribute
    -logger              generate a custom Logger type
    -api <any|attr>      the API style for the Logger's methods (default: any)
    -ctx                 add context.Context to the Logger's methods
    -h, -help            print this message and quit
```

For the description of the config file fields, see [`.slog.example.yml`](.slog.example.yml).

[1]: https://github.com/go-simpler/sloglint
