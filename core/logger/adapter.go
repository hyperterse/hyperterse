package logger

import (
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

// Re-export log level constants for backward compatibility
const (
	LogLevelError = logging.LogLevelError
	LogLevelWarn  = logging.LogLevelWarn
	LogLevelInfo  = logging.LogLevelInfo
	LogLevelDebug = logging.LogLevelDebug
)

// SetLogLevel sets the global log level (backward compatibility wrapper)
func SetLogLevel(level int) {
	logging.SetLogLevel(level)
}

// GetLogLevel returns the current global log level (backward compatibility wrapper)
func GetLogLevel() int {
	return logging.GetLogLevel()
}

// SetTagFilter sets the tag filter (backward compatibility wrapper)
func SetTagFilter(filterStr string) {
	logging.SetTagFilter(filterStr)
}

// SetLogFile enables log file streaming (backward compatibility wrapper)
func SetLogFile() (string, error) {
	return logging.SetLogFile()
}

// CloseLogFile closes the log file (backward compatibility wrapper)
func CloseLogFile() error {
	return logging.CloseLogFile()
}

// Logger wraps the new logger interface to maintain backward compatibility
type Logger struct {
	impl logging.Logger
}

// New creates a new logger instance with a tag (backward compatibility wrapper)
func New(tag string) *Logger {
	return &Logger{
		impl: logging.New(tag),
	}
}

// Error logs at ERROR level
func (l *Logger) Error(message string) {
	l.impl.Error(message)
}

// Errorf logs at ERROR level with formatting
func (l *Logger) Errorf(format string, args ...any) {
	l.impl.Errorf(format, args...)
}

// Errorln logs at ERROR level (for backward compatibility)
func (l *Logger) Errorln(args ...any) {
	l.impl.Errorln(args...)
}

// Warn logs at WARN level
func (l *Logger) Warn(message string) {
	l.impl.Warn(message)
}

// Warnf logs at WARN level with formatting
func (l *Logger) Warnf(format string, args ...any) {
	l.impl.Warnf(format, args...)
}

// Warnln logs at WARN level (for backward compatibility)
func (l *Logger) Warnln(args ...any) {
	l.impl.Warnln(args...)
}

// Info logs at INFO level
func (l *Logger) Info(message string) {
	l.impl.Info(message)
}

// Infof logs at INFO level with formatting
func (l *Logger) Infof(format string, args ...any) {
	l.impl.Infof(format, args...)
}

// Infoln logs at INFO level (for backward compatibility)
func (l *Logger) Infoln(args ...any) {
	l.impl.Infoln(args...)
}

// Success logs at INFO level but always shows regardless of log level
func (l *Logger) Success(message string) {
	l.impl.Success(message)
}

// Successf logs at INFO level but always shows regardless of log level
func (l *Logger) Successf(format string, args ...any) {
	l.impl.Successf(format, args...)
}

// Debug logs at DEBUG level
func (l *Logger) Debug(message string) {
	l.impl.Debug(message)
}

// Debugf logs at DEBUG level with formatting
func (l *Logger) Debugf(format string, args ...any) {
	l.impl.Debugf(format, args...)
}

// Debugln logs at DEBUG level (for backward compatibility)
func (l *Logger) Debugln(args ...any) {
	l.impl.Debugln(args...)
}

// PrintError logs an error (for backward compatibility)
func (l *Logger) PrintError(title string, err error) {
	l.impl.PrintError(title, err)
}

// PrintSuccess logs a success message (for backward compatibility)
func (l *Logger) PrintSuccess(message string) {
	l.impl.PrintSuccess(message)
}

// Printf logs at INFO level (for backward compatibility)
func (l *Logger) Printf(format string, args ...any) {
	l.impl.Printf(format, args...)
}

// Println logs at INFO level (for backward compatibility)
func (l *Logger) Println(args ...any) {
	l.impl.Println(args...)
}

// PrintValidationErrors logs validation errors (for backward compatibility)
func (l *Logger) PrintValidationErrors(errors []string) {
	l.impl.PrintValidationErrors(errors)
}

// Multiline logs multiple lines (for backward compatibility)
func (l *Logger) Multiline(lines []any) {
	l.impl.Multiline(lines)
}
