package slog

import (
	"fmt"
	"log"
	"os"
	"slices"
)

type LogLevel = string

const (
	INFO  LogLevel = "INFO"
	ERROR          = "ERROR"
	FATAL          = "FATAL"
)

func logFLn(logLevel LogLevel, format string, v []any) {
	log.Printf(fmt.Sprintf("%s: %s\n", logLevel, format), v...)
}

func logLn(logLevel LogLevel, v []any) {
	v = slices.Insert(v, 0, any(fmt.Sprintf("%s: ", logLevel)))
	log.Print(v...)
}

// Calls to log.Printf with INFO tag associated with the log. Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...any) {
	logFLn(INFO, format, v)
}

// Calls to log.Printf with INFO tag associated with the log. Arguments are handled in the manner of fmt.Print.
func Info(v ...any) {
	logLn(INFO, v)
}

// Calls to log.Printf with ERROR tag associated with the log. Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...any) {
	logFLn(ERROR, format, v)
}

// Calls to log.Printf with ERROR tag associated with the log. Arguments are handled in the manner of fmt.Print.
func Error(v ...any) {
	logLn(ERROR, v)
}

// Associate log with FATAL tag. Arguments are handled in the manner of fmt.Printf. Also, it calls os.Exit(1) to indicate failure
func Fatalf(format string, v ...any) {
	logFLn(FATAL, format, v)
	os.Exit(1)
}

// Associate log with FATAL tag. Arguments are handled in the manner of fmt.Print. Also, it calls os.Exit(1) to indicate failure
func Fatal(v ...any) {
	logLn(FATAL, v)
	os.Exit(1)
}
