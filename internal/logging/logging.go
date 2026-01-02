package logging

import (
	"log"
	"os"
	"strings"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelError
)

var current Level = LevelInfo

// InitFromEnv sets the log level based on LOG_LEVEL (debug|info|error).
func InitFromEnv() {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error":
		current = LevelError
	case "debug":
		current = LevelDebug
	default:
		current = LevelInfo
	}
}

func Debugf(format string, args ...interface{}) {
	if current <= LevelDebug {
		log.Printf(format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	if current <= LevelInfo {
		log.Printf(format, args...)
	}
}

func Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}
