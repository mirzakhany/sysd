package sysd

import (
	"fmt"
)

// Logger is the interface that wraps the basic logging methods
type Logger interface {
	Println(v ...any)
}

type logger struct {
	l Logger
}

// Info logs an info message
func (l *logger) Info(format string, args ...any) {
	l.l.Println("INFO", fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *logger) Error(format string, args ...any) {
	l.l.Println("ERROR", fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *logger) Warn(format string, args ...any) {
	l.l.Println("WARN", fmt.Sprintf(format, args...))
}
