package logger

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Logger provides structured logging with pretty formatting for interactive environments
type Logger struct {
	packageName string
	interactive bool
}

// New creates a new logger instance for a package
func New(packageName string) *Logger {
	return &Logger{
		packageName: packageName,
		interactive: isInteractive(),
	}
}

// isInteractive checks if the output is going to a terminal (interactive environment)
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// colorize applies color to text if in interactive mode
func (l *Logger) colorize(color, text string) string {
	if l.interactive {
		return color + text + colorReset
	}
	return text
}

// bold makes text bold if in interactive mode
func (l *Logger) bold(text string) string {
	return l.colorize(colorBold, text)
}

// red colors text red if in interactive mode
func (l *Logger) red(text string) string {
	return l.colorize(colorRed, text)
}

// yellow colors text yellow if in interactive mode
func (l *Logger) yellow(text string) string {
	return l.colorize(colorYellow, text)
}

// cyan colors text cyan if in interactive mode
func (l *Logger) cyan(text string) string {
	return l.colorize(colorCyan, text)
}

func (l *Logger) blue(text string) string {
	return l.colorize(colorBlue, text)
}

// dim makes text dimmed if in interactive mode
func (l *Logger) dim(text string) string {
	return l.colorize(colorDim, text)
}

// getPrefix returns the formatted prefix "[Hyperterse:$packageName timestamp]"
func (l *Logger) getPrefix() string {
	prefix := fmt.Sprintf("[Hyperterse:%s]", l.packageName)
	return l.dim(prefix)
}

// FormatValidationErrors formats a list of validation errors prettily
func (l *Logger) FormatValidationErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}

	var builder strings.Builder

	// Header
	header := fmt.Sprintf("%s Validation Errors (%d)", l.red("✗"), len(errors))
	builder.WriteString(l.bold(header))
	builder.WriteString("\n\n")

	// Format each error
	for i, err := range errors {
		// Number the error
		errorNum := fmt.Sprintf("%d.", i+1)
		builder.WriteString(l.cyan(errorNum))
		builder.WriteString(" ")

		// Format the error message
		// Try to extract field path (everything before "is" or "must")
		// Common patterns: "field.path is required", "field.path 'value' must be..."
		fieldPathEnd := strings.Index(err, " is ")
		if fieldPathEnd == -1 {
			fieldPathEnd = strings.Index(err, " must ")
		}
		if fieldPathEnd == -1 {
			fieldPathEnd = strings.Index(err, " ")
		}

		if fieldPathEnd > 0 {
			fieldPath := err[:fieldPathEnd]
			message := err[fieldPathEnd+1:]
			builder.WriteString(l.yellow(fieldPath))
			builder.WriteString(" ")
			builder.WriteString(message)
		} else {
			builder.WriteString(err)
		}

		if i < len(errors)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// FormatError formats a single error message prettily
func (l *Logger) FormatError(title string, err error) string {
	var builder strings.Builder

	header := fmt.Sprintf("%s %s", l.red("✗"), title)
	builder.WriteString(l.bold(header))
	builder.WriteString("\n")
	builder.WriteString(l.red(err.Error()))
	builder.WriteString("\n")

	return builder.String()
}

// FormatSuccess formats a success message prettily
func (l *Logger) FormatSuccess(message string) string {
	successIcon := l.colorize(colorCyan, "✓")
	return fmt.Sprintf("%s %s\n", successIcon, message)
}

// PrintValidationErrors prints formatted validation errors using Multiline
func (l *Logger) PrintValidationErrors(errors []string) {
	if len(errors) == 0 {
		return
	}

	// Build header line
	header := fmt.Sprintf("%s Validation Errors (%d)", l.red("✗"), len(errors))
	header = l.bold(header)

	// Build formatted error lines
	formattedErrors := make([]interface{}, len(errors))
	for i, err := range errors {
		// Use bullet point instead of number, no color formatting
		errorLine := "- " + err
		formattedErrors[i] = errorLine
	}

	// Use Multiline: header on first line with prefix, errors on subsequent lines without prefix
	args := make([]interface{}, 0, len(formattedErrors)+1)
	args = append(args, header)
	args = append(args, formattedErrors...)
	l.Multiline(args)
}

// PrintError prints a formatted error
func (l *Logger) PrintError(title string, err error) {
	formatted := l.FormatError(title, err)
	lines := strings.Split(strings.TrimRight(formatted, "\n"), "\n")
	if len(lines) > 0 {
		fmt.Print(l.getPrefix() + " " + lines[0])
		if len(lines) > 1 {
			fmt.Print("\n")
			for _, line := range lines[1:] {
				if line != "" {
					fmt.Println(line)
				} else {
					fmt.Println()
				}
			}
		} else {
			fmt.Print("\n")
		}
	}
}

// PrintSuccess prints a formatted success message
func (l *Logger) PrintSuccess(message string) {
	formatted := l.FormatSuccess(message)
	formatted = strings.TrimRight(formatted, "\n")
	fmt.Print(l.getPrefix() + " " + formatted + "\n")
}

// Printf is a convenience method that wraps fmt.Printf with prefix
func (l *Logger) Printf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	message = strings.TrimRight(message, "\n")
	fmt.Print(l.getPrefix() + " " + message + "\n")
}

// Println is a convenience method that wraps fmt.Println with prefix
func (l *Logger) Println(args ...interface{}) {
	message := fmt.Sprint(args...)
	fmt.Print(l.getPrefix() + " " + message + "\n")
}

// Multiline prints multiple lines where:
// - The first element is printed on the same line as the prefix
// - Elements 2 to N are printed on new lines WITHOUT the prefix
func (l *Logger) Multiline(lines []interface{}) {
	if len(lines) == 0 {
		return
	}

	// Print first line with prefix
	firstLine := fmt.Sprint(lines[0])
	fmt.Print(l.getPrefix() + " " + firstLine)

	// Print remaining lines without prefix
	if len(lines) > 1 {
		fmt.Print("\n")
		for _, line := range lines[1:] {
			fmt.Println(fmt.Sprint(line))
		}
	} else {
		fmt.Print("\n")
	}
}
