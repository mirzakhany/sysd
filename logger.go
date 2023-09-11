package sysd

import (
	"fmt"
)

type Logger interface {
	Println(v ...any)
}

type logger struct {
	l Logger
}

// Info logs an info message
func (l *logger) Info(format string, args ...any) {
	l.l.Println(fmt.Sprintf("INFO "+format, args))
}

// Error logs an error message
func (l *logger) Error(format string, args ...any) {
	l.l.Println(fmt.Sprintf("ERROR "+format, args))
}

// Warn logs a warning message
func (l *logger) Warn(format string, args ...any) {
	l.l.Println(fmt.Sprintf("WARN "+format, args))
}
