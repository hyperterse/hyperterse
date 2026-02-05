package logging

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
)

const (
	// Log level constants matching the original logger
	LogLevelError = 1
	LogLevelWarn  = 2
	LogLevelInfo  = 3
	LogLevelDebug = 4
)

var (
	globalLogLevel = LogLevelInfo
	logLevelMutex  sync.RWMutex

	// Tag filtering
	tagFilter      []string
	tagFilterMutex sync.RWMutex

	// Log file streaming
	logFile      *os.File
	logFileMutex sync.Mutex
	logWriter    io.Writer = os.Stdout
)

// SetLogLevel sets the global log level
func SetLogLevel(level int) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if level >= LogLevelError && level <= LogLevelDebug {
		globalLogLevel = level
		zerolog.SetGlobalLevel(convertLogLevel(level))
	}
}

// GetLogLevel returns the current global log level
func GetLogLevel() int {
	logLevelMutex.RLock()
	defer logLevelMutex.RUnlock()
	return globalLogLevel
}

// SetTagFilter sets the tag filter from a comma-separated string
func SetTagFilter(filterStr string) {
	tagFilterMutex.Lock()
	defer tagFilterMutex.Unlock()

	if filterStr == "" {
		tagFilter = nil
		return
	}

	tags := strings.Split(filterStr, ",")
	tagFilter = make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tagFilter = append(tagFilter, tag)
		}
	}
}

// shouldLogTag checks if a tag should be logged based on the filter
func shouldLogTag(tag string) bool {
	tagFilterMutex.RLock()
	defer tagFilterMutex.RUnlock()

	if len(tagFilter) == 0 {
		return true
	}

	excluded := false
	for _, filterTag := range tagFilter {
		if strings.HasPrefix(filterTag, "-") {
			excludeTag := strings.TrimPrefix(filterTag, "-")
			if tag == excludeTag || strings.HasPrefix(tag, excludeTag+":") {
				excluded = true
				break
			}
		}
	}
	if excluded {
		return false
	}

	hasInclusion := false
	for _, filterTag := range tagFilter {
		if !strings.HasPrefix(filterTag, "-") {
			hasInclusion = true
			if tag == filterTag || strings.HasPrefix(tag, filterTag+":") {
				return true
			}
		}
	}

	if hasInclusion {
		return false
	}

	return true
}

// SetLogFile enables log file streaming with auto-generated filename
func SetLogFile() (string, error) {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	logDir := "/tmp/.hyperterse/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", err
	}

	hash := generateLogFileHash()
	filename := "hyperterse-" + hash + ".log"
	filePath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}

	logFile = file
	logWriter = io.MultiWriter(os.Stdout, file)

	// Update zerolog output
	zerologOutput := logWriter
	if isInteractive() {
		zerologOutput = zerolog.ConsoleWriter{Out: logWriter, TimeFormat: "2006-01-02T15:04:05.000Z"}
	}
	log.Logger = zerolog.New(zerologOutput).With().Timestamp().Logger()

	return filePath, nil
}

// CloseLogFile closes the log file if it's open
func CloseLogFile() error {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		logWriter = os.Stdout
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
		return err
	}
	return nil
}

// generateLogFileHash generates a short hash for log filename
func generateLogFileHash() string {
	timestamp := time.Now().UnixNano()
	pid := os.Getpid()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	hashInput := fmt.Sprintf("%d-%d-%x", timestamp, pid, randomBytes)
	hash := sha256.Sum256([]byte(hashInput))

	return hex.EncodeToString(hash[:])[:8]
}

// ZerologLogger implements the Logger interface using zerolog
// Exported for use in adapter
type ZerologLogger struct {
	tag         string
	logger      zerolog.Logger
	interactive bool
}

// Logger is the interface exported from this package
type Logger = interfaces.Logger

// New creates a new logger instance with a tag
func New(tag string) Logger {
	// Check if we should log this tag
	if !shouldLogTag(tag) {
		// Return a no-op logger for filtered tags
		return &noOpLogger{}
	}

	// Create zerolog logger with tag
	logger := log.Logger.With().Str("tag", tag).Logger()

	// Configure output based on whether we're interactive
	var output io.Writer = logWriter
	if isInteractive() {
		output = zerolog.ConsoleWriter{Out: logWriter, TimeFormat: "2006-01-02T15:04:05.000Z"}
		logger = zerolog.New(output).With().Str("tag", tag).Timestamp().Logger()
	}

	return &ZerologLogger{
		tag:         tag,
		logger:      logger,
		interactive: isInteractive(),
	}
}

// isInteractive checks if the output is going to a terminal
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// convertLogLevel converts our log level to zerolog level
func convertLogLevel(level int) zerolog.Level {
	switch level {
	case LogLevelError:
		return zerolog.ErrorLevel
	case LogLevelWarn:
		return zerolog.WarnLevel
	case LogLevelInfo:
		return zerolog.InfoLevel
	case LogLevelDebug:
		return zerolog.DebugLevel
	default:
		return zerolog.InfoLevel
	}
}

// checkLogLevel checks if we should log at this level
func (l *ZerologLogger) checkLogLevel(level int) bool {
	logLevelMutex.RLock()
	shouldLog := level <= globalLogLevel
	logLevelMutex.RUnlock()
	return shouldLog
}

// Error logs at ERROR level
func (l *ZerologLogger) Error(message string) {
	if !l.checkLogLevel(LogLevelError) {
		return
	}
	l.logger.Error().Msg(message)
}

// Errorf logs at ERROR level with formatting
func (l *ZerologLogger) Errorf(format string, args ...any) {
	if !l.checkLogLevel(LogLevelError) {
		return
	}
	l.logger.Error().Msgf(format, args...)
}

// Errorln logs at ERROR level (for backward compatibility)
func (l *ZerologLogger) Errorln(args ...any) {
	if !l.checkLogLevel(LogLevelError) {
		return
	}
	l.logger.Error().Msg(fmt.Sprint(args...))
}

// Warn logs at WARN level
func (l *ZerologLogger) Warn(message string) {
	if !l.checkLogLevel(LogLevelWarn) {
		return
	}
	l.logger.Warn().Msg(message)
}

// Warnf logs at WARN level with formatting
func (l *ZerologLogger) Warnf(format string, args ...any) {
	if !l.checkLogLevel(LogLevelWarn) {
		return
	}
	l.logger.Warn().Msgf(format, args...)
}

// Warnln logs at WARN level (for backward compatibility)
func (l *ZerologLogger) Warnln(args ...any) {
	if !l.checkLogLevel(LogLevelWarn) {
		return
	}
	l.logger.Warn().Msg(fmt.Sprint(args...))
}

// Info logs at INFO level
func (l *ZerologLogger) Info(message string) {
	if !l.checkLogLevel(LogLevelInfo) {
		return
	}
	l.logger.Info().Msg(message)
}

// Infof logs at INFO level with formatting
func (l *ZerologLogger) Infof(format string, args ...any) {
	if !l.checkLogLevel(LogLevelInfo) {
		return
	}
	l.logger.Info().Msgf(format, args...)
}

// Infoln logs at INFO level (for backward compatibility)
func (l *ZerologLogger) Infoln(args ...any) {
	if !l.checkLogLevel(LogLevelInfo) {
		return
	}
	l.logger.Info().Msg(fmt.Sprint(args...))
}

// Success logs at INFO level but always shows regardless of log level
func (l *ZerologLogger) Success(message string) {
	// Use the same logger instance to respect console formatting for interactive terminals
	successLogger := l.logger.With().Str("tag", "Success").Logger()
	successLogger.Info().Msg(message)
}

// Successf logs at INFO level but always shows regardless of log level
func (l *ZerologLogger) Successf(format string, args ...any) {
	// Use the same logger instance to respect console formatting for interactive terminals
	successLogger := l.logger.With().Str("tag", "Success").Logger()
	successLogger.WithLevel(100).Msgf(format, args...)
}

// Debug logs at DEBUG level
func (l *ZerologLogger) Debug(message string) {
	if !l.checkLogLevel(LogLevelDebug) {
		return
	}
	l.logger.Debug().Msg(message)
}

// Debugf logs at DEBUG level with formatting
func (l *ZerologLogger) Debugf(format string, args ...any) {
	if !l.checkLogLevel(LogLevelDebug) {
		return
	}
	l.logger.Debug().Msgf(format, args...)
}

// Debugln logs at DEBUG level (for backward compatibility)
func (l *ZerologLogger) Debugln(args ...any) {
	if !l.checkLogLevel(LogLevelDebug) {
		return
	}
	l.logger.Debug().Msg(fmt.Sprint(args...))
}

// PrintError logs an error (for backward compatibility)
func (l *ZerologLogger) PrintError(title string, err error) {
	if err == nil {
		return
	}
	l.Errorf("%s: %v", title, err)
}

// PrintSuccess logs a success message (for backward compatibility)
func (l *ZerologLogger) PrintSuccess(message string) {
	l.Info(message)
}

// Printf logs at INFO level (for backward compatibility)
func (l *ZerologLogger) Printf(format string, args ...any) {
	l.Infof(format, args...)
}

// Println logs at INFO level (for backward compatibility)
func (l *ZerologLogger) Println(args ...any) {
	l.Infoln(args...)
}

// PrintValidationErrors logs validation errors (for backward compatibility)
func (l *ZerologLogger) PrintValidationErrors(errors []string) {
	if len(errors) == 0 {
		return
	}
	l.Errorf("Validation Errors (%d)", len(errors))
	for i, err := range errors {
		l.Errorf("  %d. %s", i+1, err)
	}
}

// Multiline logs multiple lines (for backward compatibility)
func (l *ZerologLogger) Multiline(lines []any) {
	if len(lines) == 0 {
		return
	}
	for _, line := range lines {
		lineStr := fmt.Sprint(line)
		if lineStr != "" {
			l.Info(lineStr)
		}
	}
}

// noOpLogger is a no-op logger for filtered tags
type noOpLogger struct{}

func (n *noOpLogger) Error(string)                   {}
func (n *noOpLogger) Errorf(string, ...any)          {}
func (n *noOpLogger) Errorln(...any)                 {}
func (n *noOpLogger) Warn(string)                    {}
func (n *noOpLogger) Warnf(string, ...any)           {}
func (n *noOpLogger) Warnln(...any)                  {}
func (n *noOpLogger) Info(string)                    {}
func (n *noOpLogger) Infof(string, ...any)           {}
func (n *noOpLogger) Infoln(...any)                  {}
func (n *noOpLogger) Success(string)                 {}
func (n *noOpLogger) Successf(string, ...any)        {}
func (n *noOpLogger) Debug(string)                   {}
func (n *noOpLogger) Debugf(string, ...any)          {}
func (n *noOpLogger) Debugln(...any)                 {}
func (n *noOpLogger) PrintError(string, error)       {}
func (n *noOpLogger) PrintSuccess(string)            {}
func (n *noOpLogger) Printf(string, ...any)          {}
func (n *noOpLogger) Println(...any)                 {}
func (n *noOpLogger) PrintValidationErrors([]string) {}
func (n *noOpLogger) Multiline([]any)                {}
