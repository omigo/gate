package spdy

import (
	golog "log"
)

type Logger struct {
	Level byte
}

const (
	TRACE byte = iota + 1
	DEBUG
	INFO
	NOTICE
	WARN
	ERROR
	FATAL
)

var log *Logger

func init() {
	log = &Logger{
		Level: NOTICE,
	}
}

func GetLogger() *Logger {
	return log
}

func (log *Logger) SetLevel(level byte) {
	log.Level = level
}

func (log *Logger) Trace(format string, msg ...interface{}) {
	if log.Level <= TRACE {
		golog.Printf(format, msg...)
	}
}

func (log *Logger) Debug(format string, msg ...interface{}) {
	if log.Level <= DEBUG {
		golog.Printf(format, msg...)
	}
}

func (log *Logger) Info(format string, msg ...interface{}) {
	if log.Level <= INFO {
		golog.Printf(format, msg...)
	}
}

func (log *Logger) Notice(format string, msg ...interface{}) {
	if log.Level <= NOTICE {
		golog.Printf(format, msg...)
	}
}

func (log *Logger) Warn(format string, msg ...interface{}) {
	if log.Level <= WARN {
		golog.Printf(format, msg...)
	}
}

func (log *Logger) Error(format string, msg ...interface{}) {
	if log.Level <= ERROR {
		golog.Printf(format, msg...)
	}
}

func (log *Logger) Fatal(format string, msg ...interface{}) {
	if log.Level <= FATAL {
		golog.Printf(format, msg...)
	}
}
