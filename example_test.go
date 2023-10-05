package main

// NOTE: replace "go run main.go sloggen.go" with "go run go-simpler.org/sloggen@<version>" in your project.

// using flags:
//go:generate go run main.go sloggen.go -pkg=example -i=time -l=info:0 -l=alert:1 -c=request_id -a=user_id:int -a=created_at:time.Time -a=err:error -logger -api=attr -ctx

// using config (see .slog.example.yml):
//go:generate go run main.go sloggen.go -config=.slog.example.yml
