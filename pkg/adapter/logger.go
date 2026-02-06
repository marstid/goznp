package adapter

import (
	"context"
	"fmt"
	"log/slog"
)

// Logger defines the logging interface used by the adapter.
// Implementations can be provided to control log output.
type Logger interface {
	// Debug logs a debug message.
	Debugf(format string, args ...interface{})

	// Info logs an informational message.
	Infof(format string, args ...interface{})

	// Warn logs a warning message.
	Warnf(format string, args ...interface{})

	// Error logs an error message.
	Errorf(format string, args ...interface{})
}

// DefaultLogger is a no-op logger that discards all log messages.
// This is used when no logger is configured.
type DefaultLogger struct{}

func (l *DefaultLogger) Debugf(_ string, _ ...interface{}) {}
func (l *DefaultLogger) Infof(_ string, _ ...interface{})  {}
func (l *DefaultLogger) Warnf(_ string, _ ...interface{})  {}
func (l *DefaultLogger) Errorf(_ string, _ ...interface{}) {}

// Ensure DefaultLogger implements Logger interface.
var _ Logger = (*DefaultLogger)(nil)

// SlogLogger adapts a [log/slog.Logger] to the [Logger] interface.
// This allows using Go's standard structured logging with the adapter:
//
//	adapter.New(adapter.WithLogger(adapter.NewSlogLogger(slog.Default())))
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a [Logger] that delegates to the given [slog.Logger].
// If logger is nil, [slog.Default] is used.
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogLogger{logger: logger}
}

func (l *SlogLogger) Debugf(format string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelDebug, fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Infof(format string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelInfo, fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Warnf(format string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelWarn, fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Errorf(format string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
}

// Ensure SlogLogger implements Logger interface.
var _ Logger = (*SlogLogger)(nil)
