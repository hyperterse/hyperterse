package logger

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

	"golang.org/x/term"
)

// Log level constants
const (
	LogLevelError = 1
	LogLevelWarn  = 2
	LogLevelInfo  = 3
	LogLevelDebug = 4
)

// ANSI color codes
const (
	colorReset    = "\033[0m"
	colorRed      = "\033[31m"
	colorYellow   = "\033[33m"
	colorCyan     = "\033[36m"
	colorBlue     = "\033[34m"
	colorGreen    = "\033[32m"
	colorDim      = "\033[2m"
	colorWhite    = "\033[37m"
	colorBold     = "\033[1m"
	colorSuperDim = "\033[2m\033[90m" // Dim + dark gray for super dimmed text
	bgRed         = "\033[41m"
	bgYellow      = "\033[43m"
	bgCyan        = "\033[46m"
	bgBlue        = "\033[44m"
	bgGreen       = "\033[42m"
	bgDim         = "\033[100m" // Dark gray background for DEBUG
)

var (
	globalLogLevel = LogLevelInfo // Default to INFO
	logLevelMutex  sync.RWMutex

	// Tag filtering
	tagFilter      []string // Empty means no filter, otherwise list of tags to include/exclude
	tagFilterMutex sync.RWMutex

	// Log file streaming
	logFile      *os.File
	logFileMutex sync.Mutex
	logWriter    io.Writer = os.Stdout // Default to stdout, can be MultiWriter
)

// SetLogLevel sets the global log level
func SetLogLevel(level int) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if level >= LogLevelError && level <= LogLevelDebug {
		globalLogLevel = level
	}
}

// GetLogLevel returns the current global log level
func GetLogLevel() int {
	logLevelMutex.RLock()
	defer logLevelMutex.RUnlock()
	return globalLogLevel
}

// SetTagFilter sets the tag filter from a comma-separated string
// Supports inclusion (tag1,tag2) and exclusion (-tag1) patterns
// Empty string clears the filter
func SetTagFilter(filterStr string) {
	tagFilterMutex.Lock()
	defer tagFilterMutex.Unlock()

	if filterStr == "" {
		tagFilter = nil
		return
	}

	// Split by comma and trim spaces
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

	// No filter means log everything
	if len(tagFilter) == 0 {
		return true
	}

	// Check for exclusion patterns (tags starting with -)
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

	// Check for inclusion patterns
	// If any inclusion patterns exist, tag must match at least one
	hasInclusion := false
	for _, filterTag := range tagFilter {
		if !strings.HasPrefix(filterTag, "-") {
			hasInclusion = true
			if tag == filterTag || strings.HasPrefix(tag, filterTag+":") {
				return true
			}
		}
	}

	// If we have inclusion patterns but none matched, don't log
	if hasInclusion {
		return false
	}

	// No inclusion patterns, only exclusions (which we already checked)
	return true
}

// SetLogFile enables log file streaming with auto-generated filename
// Returns the file path if successful, or error if failed
func SetLogFile() (string, error) {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	// Create log directory
	logDir := "/tmp/.hyperterse/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Generate unique filename hash
	hash := generateLogFileHash()
	filename := fmt.Sprintf("hyperterse-%s.log", hash)
	filePath := filepath.Join(logDir, filename)

	// Open file for writing (create if not exists, append if exists)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}

	logFile = file

	// Create MultiWriter to write to both stdout and file
	logWriter = io.MultiWriter(os.Stdout, file)

	return filePath, nil
}

// generateLogFileHash generates a short hash for log filename
func generateLogFileHash() string {
	// Combine timestamp, PID, and random bytes
	timestamp := time.Now().UnixNano()
	pid := os.Getpid()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	// Create hash input
	hashInput := fmt.Sprintf("%d-%d-%x", timestamp, pid, randomBytes)
	hash := sha256.Sum256([]byte(hashInput))

	// Return first 8 characters as hex
	return hex.EncodeToString(hash[:])[:8]
}

// CloseLogFile closes the log file if it's open
func CloseLogFile() error {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		logWriter = os.Stdout
		return err
	}
	return nil
}

// Logger provides structured logging with Android Logcat-inspired format
type Logger struct {
	tag         string
	interactive bool
}

// New creates a new logger instance with a tag
func New(tag string) *Logger {
	return &Logger{
		tag:         tag,
		interactive: isInteractive(),
	}
}

// isInteractive checks if the output is going to a terminal
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// formatLogEntry formats a log entry according to the new format
// Format: TIMESTAMP  LEVEL  TAG: message
func (l *Logger) formatLogEntry(level int, levelChar string, levelColor string, levelBgColor string, message string) string {
	// Get current timestamp in ISO8601 format with milliseconds
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	// Format timestamp - super dimmed out
	timestampStr := timestamp
	if l.interactive {
		timestampStr = colorSuperDim + timestamp + colorReset
	}

	// Format level tag with background color and white foreground, with spaces before and after
	levelStr := " " + levelChar + " "
	if l.interactive {
		levelStr = levelBgColor + colorWhite + colorBold + " " + levelChar + " " + colorReset
	}

	// Format tag with the same color as the log level
	tagStr := l.tag
	if l.interactive {
		tagStr = levelColor + l.tag + colorReset
	}

	// For ERROR and WARN, color the entire log line
	if l.interactive && (level == LogLevelError || level == LogLevelWarn) {
		// Color the entire message
		coloredMessage := levelColor + message + colorReset
		return fmt.Sprintf("%s  %s  %s: %s\n", timestampStr, levelStr, tagStr, coloredMessage)
	}

	// For INFO and DEBUG, color the tag and message with level color
	return fmt.Sprintf("%s  %s  %s: %s\n", timestampStr, levelStr, tagStr, message)
}

// writeLog writes a log entry if it passes level and tag filters
// If message contains newlines, each line is logged separately with the tag
func (l *Logger) writeLog(level int, levelChar string, levelColor string, levelBgColor string, message string) {
	// Check log level
	logLevelMutex.RLock()
	shouldLog := level <= globalLogLevel
	logLevelMutex.RUnlock()

	if !shouldLog {
		return
	}

	// Check tag filter
	if !shouldLogTag(l.tag) {
		return
	}

	l.writeLogUnfiltered(level, levelChar, levelColor, levelBgColor, message)
}

// writeLogUnfiltered writes a log entry without checking log level or tag filters
// Used for critical messages that should always be shown
func (l *Logger) writeLogUnfiltered(level int, levelChar string, levelColor string, levelBgColor string, message string) {
	// Split message by newlines and log each line separately
	lines := strings.Split(message, "\n")
	for i, line := range lines {
		// Skip empty lines unless it's the first line
		if i > 0 && strings.TrimSpace(line) == "" {
			continue
		}

		// Format and write log entry
		logEntry := l.formatLogEntry(level, levelChar, levelColor, levelBgColor, line)

		logFileMutex.Lock()
		writer := logWriter
		logFileMutex.Unlock()

		writer.Write([]byte(logEntry))
	}
}

// Error logs at ERROR level
func (l *Logger) Error(message string) {
	l.writeLog(LogLevelError, "E", colorRed, bgRed, message)
}

// Errorf logs at ERROR level with formatting
func (l *Logger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

// Errorln logs at ERROR level (for backward compatibility)
func (l *Logger) Errorln(args ...any) {
	l.Error(fmt.Sprint(args...))
}

// Warn logs at WARN level
func (l *Logger) Warn(message string) {
	l.writeLog(LogLevelWarn, "W", colorYellow, bgYellow, message)
}

// Warnf logs at WARN level with formatting
func (l *Logger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Warnln logs at WARN level (for backward compatibility)
func (l *Logger) Warnln(args ...any) {
	l.Warn(fmt.Sprint(args...))
}

// Info logs at INFO level
func (l *Logger) Info(message string) {
	l.writeLog(LogLevelInfo, "I", colorCyan, bgCyan, message)
}

// Success logs at INFO level but always shows regardless of log level, with "success" as the tag
// Uses green background with white foreground
func (l *Logger) Success(message string) {
	// Create a temporary logger with "success" tag
	successLogger := &Logger{tag: "Success", interactive: l.interactive}
	successLogger.writeLogUnfiltered(LogLevelInfo, "✔", colorGreen, bgGreen, message)
}

// Successf logs at INFO level but always shows regardless of log level, with "success" as the tag
func (l *Logger) Successf(format string, args ...any) {
	l.Success(fmt.Sprintf(format, args...))
}

// Infof logs at INFO level with formatting
func (l *Logger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

// Infoln logs at INFO level (for backward compatibility)
func (l *Logger) Infoln(args ...any) {
	l.Info(fmt.Sprint(args...))
}

// Debug logs at DEBUG level
func (l *Logger) Debug(message string) {
	l.writeLog(LogLevelDebug, "D", colorDim, bgDim, message)
}

// Debugf logs at DEBUG level with formatting
func (l *Logger) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Debugln logs at DEBUG level (for backward compatibility)
func (l *Logger) Debugln(args ...any) {
	l.Debug(fmt.Sprint(args...))
}

// PrintError logs an error (for backward compatibility)
// If the error message contains newlines, each line is logged separately with the tag
func (l *Logger) PrintError(title string, err error) {
	if err == nil {
		return
	}
	l.Errorf("%s: %v", title, err)
}

// PrintSuccess logs a success message at INFO level (for backward compatibility)
func (l *Logger) PrintSuccess(message string) {
	l.Info(message)
}

// Printf logs at INFO level (for backward compatibility)
func (l *Logger) Printf(format string, args ...any) {
	l.Infof(format, args...)
}

// Println logs at INFO level (for backward compatibility)
func (l *Logger) Println(args ...any) {
	l.Infoln(args...)
}

// PrintValidationErrors logs validation errors (for backward compatibility)
func (l *Logger) PrintValidationErrors(errors []string) {
	if len(errors) == 0 {
		return
	}
	l.Errorf("Validation Errors (%d)", len(errors))
	for i, err := range errors {
		l.Errorf("  %d. %s", i+1, err)
	}
}

// Multiline logs multiple lines (for backward compatibility)
// All lines are logged with the same tag
func (l *Logger) Multiline(lines []any) {
	if len(lines) == 0 {
		return
	}

	// Log all lines with tag
	for _, line := range lines {
		lineStr := fmt.Sprint(line)
		if lineStr != "" {
			l.Info(lineStr)
		}
	}
}
