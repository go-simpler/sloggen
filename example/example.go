// Code generated by go-simpler.org/slog-gen. DO NOT EDIT.

package example

import "log/slog"
import "time"

const LevelDebug = slog.Level(-4)
const LevelError = slog.Level(8)
const LevelInfo = slog.Level(0)
const LevelTrace = slog.Level(-8)
const LevelWarn = slog.Level(4)

const RequestId = "request_id"

func CreatedAt(value time.Time) slog.Attr {
	return slog.Time("created_at", value)
}

func Err(value error) slog.Attr {
	return slog.Any("err", value)
}

func UserId(value int) slog.Attr {
	return slog.Int("user_id", value)
}
