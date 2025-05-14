package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents the logging level
type LogLevel string

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel LogLevel = "debug"
	// InfoLevel is the default logging priority.
	InfoLevel LogLevel = "info"
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel LogLevel = "warn"
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel LogLevel = "error"
)

// NewDevelopment creates a new development logger that writes to stdout
// with a human-readable format.
func NewDevelopment(level LogLevel) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Set the log level
	var zapLevel zapcore.Level
	switch level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	return config.Build()
}

// MustNewDevelopment creates a new development logger and panics if an error occurs.
func MustNewDevelopment(level LogLevel) *zap.Logger {
	logger, err := NewDevelopment(level)
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	return logger
}

// NewNop returns a no-op logger that does nothing.
func NewNop() *zap.Logger {
	return zap.NewNop()
}

// Global logger instance
var (
	globalLogger *zap.Logger
)

// Init initializes the global logger with development configuration.
func Init(level LogLevel) error {
	logger, err := NewDevelopment(level)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// MustInit initializes the global logger and panics if an error occurs.
func MustInit(level LogLevel) {
	if err := Init(level); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
}

// L returns the global logger instance.
// If the global logger is not initialized, it returns a no-op logger.
func L() *zap.Logger {
	if globalLogger == nil {
		return NewNop()
	}
	return globalLogger
}

// Sync flushes any buffered log entries.
// Applications should call Sync before exiting.
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
} 