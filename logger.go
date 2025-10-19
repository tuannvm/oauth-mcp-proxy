package oauth

import "log"

// Logger interface for pluggable logging.
// Implement this interface to integrate oauth-mcp-proxy with your application's
// logging system (e.g., zap, logrus, slog). If not provided in Config, a default
// logger using log.Printf will be used.
//
// Example:
//
//	type MyLogger struct{ logger *zap.Logger }
//	func (l *MyLogger) Info(msg string, args ...interface{}) {
//	    l.logger.Sugar().Infof(msg, args...)
//	}
//	// ... implement Debug, Warn, Error
//
//	cfg := &oauth.Config{
//	    Provider: "okta",
//	    Logger: &MyLogger{logger: zapLogger},
//	}
type Logger interface {
	Debug(msg string, args ...interface{}) // Debug-level logging for detailed troubleshooting
	Info(msg string, args ...interface{})  // Info-level logging for normal OAuth operations
	Warn(msg string, args ...interface{})  // Warn-level logging for security violations
	Error(msg string, args ...interface{}) // Error-level logging for OAuth failures
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
