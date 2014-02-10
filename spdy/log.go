package spdy

import (
	"fmt"
	golog "log"
	"runtime"
	"strings"
)

type Logger struct {
	Level byte
}

const (
	TRACE byte = iota + 1
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

var log *Logger

func init() {
	golog.SetFlags(golog.Ltime)
	log = &Logger{
		Level: INFO,
	}
}

func GetLogger() *Logger {
	return log
}

func (log *Logger) SetLevel(level byte) {
	log.Level = level
}

func (log *Logger) TraceEnabled() bool {
	return log.Level <= TRACE
}

func (log *Logger) Trace(format string, msg ...interface{}) {
	if log.Level <= TRACE {
		printLog(format, msg...)
	}
}

func (log *Logger) DebugEnabled() bool {
	return log.Level <= DEBUG
}

func (log *Logger) Debug(format string, msg ...interface{}) {
	if log.Level <= DEBUG {
		printLog(format, msg...)
	}
}

func (log *Logger) Info(format string, msg ...interface{}) {
	if log.Level <= INFO {
		printLog(format, msg...)
	}
}

func (log *Logger) Warn(format string, msg ...interface{}) {
	if log.Level <= WARN {
		printLog(format, msg...)
	}
}

func (log *Logger) Error(format string, msg ...interface{}) {
	if log.Level <= ERROR {
		printLog(format, msg...)
	}
}

func (log *Logger) Fatal(format string, msg ...interface{}) {
	if log.Level <= FATAL {
		printLog(format, msg...)
	}
}

func printLog(format string, msg ...interface{}) {
	// Determine caller func

	_, file, lineno, ok := runtime.Caller(2)
	if ok {
		format = fmt.Sprintf("%s(%d) - ", file[strings.LastIndex(file, "/")+1:], lineno) + format

		//		funcName := runtime.FuncForPC(pc).Name()
		//		last1 := strings.LastIndex(funcName, ".")
		//		last2 := strings.LastIndex(funcName[:last1], ".")
		//		last2s := strings.LastIndex(funcName[:last1], "/")
		//		if last2 < last2s {
		//			last2 = last2s
		//		}
		//		format = fmt.Sprintf("%s(%d) %s - ", file[strings.LastIndex(file, "/")+1:], lineno, funcName[last2+1:]) + format
	}
	golog.Printf(format, msg...)
}
