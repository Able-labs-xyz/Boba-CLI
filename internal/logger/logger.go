package logger

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

var Log *log.Logger

func Init(level string) {
	Log = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
	})

	switch strings.ToLower(level) {
	case "debug":
		Log.SetLevel(log.DebugLevel)
	case "warn":
		Log.SetLevel(log.WarnLevel)
	case "error":
		Log.SetLevel(log.ErrorLevel)
	default:
		Log.SetLevel(log.InfoLevel)
	}
}

func Debug(msg string, keyvals ...any) {
	if Log == nil {
		Init("info")
	}
	Log.Debug(msg, keyvals...)
}

func Info(msg string, keyvals ...any) {
	if Log == nil {
		Init("info")
	}
	Log.Info(msg, keyvals...)
}

func Warn(msg string, keyvals ...any) {
	if Log == nil {
		Init("info")
	}
	Log.Warn(msg, keyvals...)
}

func Error(msg string, keyvals ...any) {
	if Log == nil {
		Init("info")
	}
	Log.Error(msg, keyvals...)
}

func Fatal(msg string, keyvals ...any) {
	if Log == nil {
		Init("info")
	}
	Log.Fatal(msg, keyvals...)
}
