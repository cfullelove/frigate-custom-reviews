package logger

import (
	"fmt"
	"log"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	NOTICE
	WARN
	ERROR
	FATAL
)

var currentLevel Level = INFO

func SetLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "NOTICE":
		currentLevel = NOTICE
	case "WARN", "WARNING":
		currentLevel = WARN
	case "ERROR":
		currentLevel = ERROR
	case "FATAL":
		currentLevel = FATAL
	default:
		currentLevel = INFO
	}
}

func output(level Level, prefix string, format string, v ...interface{}) {
	if currentLevel <= level {
		msg := fmt.Sprintf(format, v...)
		log.Printf("[%s] %s", prefix, msg)
	}
}

func outputLn(level Level, prefix string, v ...interface{}) {
	if currentLevel <= level {
		msg := fmt.Sprint(v...)
		log.Printf("[%s] %s", prefix, msg)
	}
}

func Debug(v ...interface{}) {
	outputLn(DEBUG, "DEBUG", v...)
}

func Debugf(format string, v ...interface{}) {
	output(DEBUG, "DEBUG", format, v...)
}

func Info(v ...interface{}) {
	outputLn(INFO, "INFO", v...)
}

func Infof(format string, v ...interface{}) {
	output(INFO, "INFO", format, v...)
}

func Notice(v ...interface{}) {
	outputLn(NOTICE, "NOTICE", v...)
}

func Noticef(format string, v ...interface{}) {
	output(NOTICE, "NOTICE", format, v...)
}

func Warn(v ...interface{}) {
	outputLn(WARN, "WARN", v...)
}

func Warnf(format string, v ...interface{}) {
	output(WARN, "WARN", format, v...)
}

func Error(v ...interface{}) {
	outputLn(ERROR, "ERROR", v...)
}

func Errorf(format string, v ...interface{}) {
	output(ERROR, "ERROR", format, v...)
}

func Fatal(v ...interface{}) {
	log.Fatalf("[FATAL] %s", fmt.Sprint(v...))
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}
