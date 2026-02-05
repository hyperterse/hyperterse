package interfaces

// Logger defines the interface for logging operations
type Logger interface {
	// Error logs at ERROR level
	Error(message string)
	// Errorf logs at ERROR level with formatting
	Errorf(format string, args ...any)
	// Errorln logs at ERROR level (for backward compatibility)
	Errorln(args ...any)

	// Warn logs at WARN level
	Warn(message string)
	// Warnf logs at WARN level with formatting
	Warnf(format string, args ...any)
	// Warnln logs at WARN level (for backward compatibility)
	Warnln(args ...any)

	// Info logs at INFO level
	Info(message string)
	// Infof logs at INFO level with formatting
	Infof(format string, args ...any)
	// Infoln logs at INFO level (for backward compatibility)
	Infoln(args ...any)

	// Success logs at INFO level but always shows regardless of log level
	Success(message string)
	// Successf logs at INFO level but always shows regardless of log level
	Successf(format string, args ...any)

	// Debug logs at DEBUG level
	Debug(message string)
	// Debugf logs at DEBUG level with formatting
	Debugf(format string, args ...any)
	// Debugln logs at DEBUG level (for backward compatibility)
	Debugln(args ...any)

	// PrintError logs an error (for backward compatibility)
	PrintError(title string, err error)
	// PrintSuccess logs a success message (for backward compatibility)
	PrintSuccess(message string)
	// Printf logs at INFO level (for backward compatibility)
	Printf(format string, args ...any)
	// Println logs at INFO level (for backward compatibility)
	Println(args ...any)
	// PrintValidationErrors logs validation errors (for backward compatibility)
	PrintValidationErrors(errors []string)
	// Multiline logs multiple lines (for backward compatibility)
	Multiline(lines []any)
}
