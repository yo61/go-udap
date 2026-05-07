package udap

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	SetLevel(level LogLevel)
}

// StructuredLogger implements the Logger interface
type StructuredLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger() *StructuredLogger {
	return &StructuredLogger{
		level:  LogLevelInfo, // Default to Info level
		logger: log.New(os.Stderr, "", 0),
	}
}

// SetLevel sets the minimum log level
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.level = level
}

// log formats and outputs a log message with structured key-value pairs
func (l *StructuredLogger) log(level LogLevel, msg string, keysAndValues ...any) {
	if level < l.level {
		return
	}

	// Build the log message
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", level.String()))
	parts = append(parts, fmt.Sprintf("[%s]", time.Now().Format("15:04:05")))
	parts = append(parts, msg)

	// Add structured key-value pairs
	if len(keysAndValues) > 0 {
		var kvPairs []string
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				key := fmt.Sprintf("%v", keysAndValues[i])
				value := fmt.Sprintf("%v", keysAndValues[i+1])
				kvPairs = append(kvPairs, fmt.Sprintf("%s=%s", key, value))
			}
		}
		if len(kvPairs) > 0 {
			parts = append(parts, fmt.Sprintf("(%s)", strings.Join(kvPairs, " ")))
		}
	}

	l.logger.Println(strings.Join(parts, " "))
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, keysAndValues ...any) {
	l.log(LogLevelDebug, msg, keysAndValues...)
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, keysAndValues ...any) {
	l.log(LogLevelInfo, msg, keysAndValues...)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, keysAndValues ...any) {
	l.log(LogLevelWarn, msg, keysAndValues...)
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, keysAndValues ...any) {
	l.log(LogLevelError, msg, keysAndValues...)
}

// NoOpLogger is a logger that does nothing (for testing or when logging is disabled)
type NoOpLogger struct{}

func (NoOpLogger) Debug(msg string, keysAndValues ...any) {}
func (NoOpLogger) Info(msg string, keysAndValues ...any)  {}
func (NoOpLogger) Warn(msg string, keysAndValues ...any)  {}
func (NoOpLogger) Error(msg string, keysAndValues ...any) {}
func (NoOpLogger) SetLevel(level LogLevel)                {}

// NewNoOpLogger creates a no-op logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}
