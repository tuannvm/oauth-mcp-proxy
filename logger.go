package oauth

import "log"

// Logger interface for pluggable logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// defaultLogger implements Logger using standard library log
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] "+msg, args...)
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}
