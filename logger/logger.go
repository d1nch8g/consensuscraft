package logger

import (
	"fmt"
	"log"
	"time"
)

func init() {
	// Disable the standard log package's timestamp and prefix
	log.SetFlags(0)
}

// formatMessage formats a message with the bedrock server style timestamp and log level
func formatMessage(level, message string) string {
	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05")
	milliseconds := now.Nanosecond() / 1000000
	return fmt.Sprintf("[%s:%03d %s] [CONSENSUSCRAFT] %s", timestamp, milliseconds, level, message)
}

// Info logs an info level message
func Info(v ...interface{}) {
	message := fmt.Sprint(v...)
	log.Print(formatMessage("INFO", message))
}

// Infof logs a formatted info level message
func Infof(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	log.Print(formatMessage("INFO", message))
}

// Error logs an error level message
func Error(v ...interface{}) {
	message := fmt.Sprint(v...)
	log.Print(formatMessage("ERROR", message))
}

// Errorf logs a formatted error level message
func Errorf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	log.Print(formatMessage("ERROR", message))
}

// Warn logs a warning level message
func Warn(v ...interface{}) {
	message := fmt.Sprint(v...)
	log.Print(formatMessage("WARN", message))
}

// Warnf logs a formatted warning level message
func Warnf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	log.Print(formatMessage("WARN", message))
}

// Debug logs a debug level message
func Debug(v ...interface{}) {
	message := fmt.Sprint(v...)
	log.Print(formatMessage("DEBUG", message))
}

// Debugf logs a formatted debug level message
func Debugf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	log.Print(formatMessage("DEBUG", message))
}

// Legacy functions for backward compatibility - default to INFO level
func Print(v ...interface{}) {
	Info(v...)
}

func Printf(format string, v ...interface{}) {
	Infof(format, v...)
}

func Println(v ...interface{}) {
	Info(v...)
}
