package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents log levels
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging
type Logger struct {
	mu       sync.Mutex
	level    Level
	format   string
	output   io.Writer
	fields   map[string]interface{}
	filePath string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp    string                 `json:"timestamp"`
	Level        string                 `json:"level"`
	Message      string                 `json:"message"`
	Caller       string                 `json:"caller,omitempty"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
	User         string                 `json:"user,omitempty"`
	Action       string                 `json:"action,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the default logger
func Init(level, format, output, filePath string) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(level, format, output, filePath)
	})
	return err
}

// NewLogger creates a new logger instance
func NewLogger(level, format, output, filePath string) (*Logger, error) {
	l := &Logger{
		level:  parseLevel(level),
		format: format,
		fields: make(map[string]interface{}),
	}

	switch output {
	case "file":
		if filePath == "" {
			return nil, fmt.Errorf("file path required for file output")
		}
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		l.output = file
		l.filePath = filePath
	case "stdout":
		l.output = os.Stdout
	default:
		l.output = os.Stdout
	}

	return l, nil
}

// parseLevel converts string level to Level type
func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		format: l.format,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   fmt.Sprintf(msg, args...),
		Fields:    l.fields,
	}

	// Add caller info
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", file, line)
	}

	// Extract special fields
	if user, ok := l.fields["user"].(string); ok {
		entry.User = user
	}
	if action, ok := l.fields["action"].(string); ok {
		entry.Action = action
	}
	if resourceType, ok := l.fields["resource_type"].(string); ok {
		entry.ResourceType = resourceType
	}
	if resourceID, ok := l.fields["resource_id"].(string); ok {
		entry.ResourceID = resourceID
	}
	if status, ok := l.fields["status"].(string); ok {
		entry.Status = status
	}
	if errStr, ok := l.fields["error"].(string); ok {
		entry.Error = errStr
	}

	var output string
	if l.format == "json" {
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = fmt.Sprintf("[%s] %s - %s", entry.Level, entry.Timestamp, entry.Message)
		if len(l.fields) > 0 {
			output += fmt.Sprintf(" | fields: %v", l.fields)
		}
	}

	fmt.Fprintln(l.output, output)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DEBUG, msg, args...)
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(INFO, msg, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WARN, msg, args...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ERROR, msg, args...)
}

// Fatal logs at fatal level and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FATAL, msg, args...)
}

// Audit logs an audit entry with user action details
func (l *Logger) Audit(user, action, resourceType, resourceID, status string, err error) {
	fields := map[string]interface{}{
		"user":          user,
		"action":        action,
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"status":        status,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	l.WithFields(fields).Info("Audit: %s performed %s on %s/%s", user, action, resourceType, resourceID)
}

// Default logger functions

// Debug logs at debug level using default logger
func Debug(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, args...)
	}
}

// Info logs at info level using default logger
func Info(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, args...)
	}
}

// Warn logs at warn level using default logger
func Warn(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, args...)
	}
}

// Error logs at error level using default logger
func Error(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, args...)
	}
}

// Fatal logs at fatal level using default logger and exits
func Fatal(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatal(msg, args...)
	}
}

// WithFields returns a new logger with fields using default logger
func WithFields(fields map[string]interface{}) *Logger {
	if defaultLogger != nil {
		return defaultLogger.WithFields(fields)
	}
	return nil
}

// Audit logs an audit entry using default logger
func Audit(user, action, resourceType, resourceID, status string, err error) {
	if defaultLogger != nil {
		defaultLogger.Audit(user, action, resourceType, resourceID, status, err)
	}
}

// GetDefault returns the default logger
func GetDefault() *Logger {
	return defaultLogger
}

