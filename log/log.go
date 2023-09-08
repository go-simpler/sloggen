// Code generated by go-simpler.org/slog-gen. DO NOT EDIT.

package log

import "log/slog"
import "time"

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