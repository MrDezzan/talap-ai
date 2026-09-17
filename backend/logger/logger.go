// Package logger provides small structured-logging helpers around Go's
// standard-library log/slog, used throughout the backend instead of the
// plain "log" package. It emits JSON logs (with timestamps) when running
// in production (GIN_MODE=release), and human-readable text logs otherwise.
package logger

import (
	"log/slog"
	"os"
)

var base *slog.Logger

func init() {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		AddSource: false,
	}
	if os.Getenv("GIN_MODE") == "release" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	base = slog.New(handler)
}

// Info logs an informational message with optional key-value args.
func Info(msg string, args ...any) {
	base.Info(msg, args...)
}

// Warn logs a warning message with optional key-value args.
func Warn(msg string, args ...any) {
	base.Warn(msg, args...)
}

// Error logs an error message with optional key-value args.
func Error(msg string, args ...any) {
	base.Error(msg, args...)
}

// Fatal logs an error message and then terminates the process, mirroring
// the standard library's log.Fatal behavior.
func Fatal(msg string, args ...any) {
	base.Error(msg, args...)
	os.Exit(1)
}
