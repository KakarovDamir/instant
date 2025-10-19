// Package logger provides structured logging configuration for the application.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// LogFormat represents the output format for logs
type LogFormat string

const (
	// FormatJSON outputs logs in JSON format (production default)
	FormatJSON LogFormat = "json"
	// FormatText outputs logs in human-readable text format (development default)
	FormatText LogFormat = "text"
)

// New creates a new structured logger based on environment configuration.
// It reads LOG_LEVEL and LOG_FORMAT from environment variables.
//
// LOG_LEVEL options: debug, info, warn, error (default: info)
// LOG_FORMAT options: json, text (default: json)
func New() *slog.Logger {
	level := getLogLevel()
	format := getLogFormat()

	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: level,
		// Add source location for error and warn levels
		AddSource: level <= slog.LevelWarn,
	}

	switch format {
	case FormatText:
		handler = slog.NewTextHandler(os.Stdout, opts)
	case FormatJSON:
		fallthrough
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// getLogLevel parses LOG_LEVEL environment variable and returns the corresponding slog.Level
func getLogLevel() slog.Level {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))

	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default to info
	}
}

// getLogFormat parses LOG_FORMAT environment variable and returns the corresponding format
func getLogFormat() LogFormat {
	formatStr := strings.ToLower(os.Getenv("LOG_FORMAT"))

	switch formatStr {
	case "text":
		return FormatText
	case "json":
		return FormatJSON
	default:
		return FormatJSON // Default to JSON for production readiness
	}
}

// SetDefault sets the given logger as the default slog logger
func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}
