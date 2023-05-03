package webapp

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
)

type Loglevel uint8

const (
	DebugLevel Loglevel = iota
	InfoLevel
	WarningLevel
	ErrorLevel
	FatalLevel
)

func SetupLogging() {
	logfile := path.Join(Config.LogDirectory, appName+".log")
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		Logf(FatalLevel, "error opening logfile: %v", err)
	}
	//defer f.Close()

	log.SetOutput(f)
}

func (l Loglevel) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return ""
	case WarningLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal error"
	}
	return fmt.Sprintf("unknown log level: %d", l)
}

// Logf is a wrapper for log.Printf or log.Fatalf if the log level is set to FatalLevel.
// The log message is ignored when the level is below the configured log level from Config.LogLevel.
// If level is set to ErrorLevel or FatalLevel the file name and line of the source file is included and in case of the
// FatalLevel a stack trace added.
// If the configured log level in Config.LogLevel is set to DebugLevel the stack trace will be added to ErrorLevel entries
// as well.
func Logf(level Loglevel, format string, msg ...any) {
	if level < Config.LogLevel {
		return
	}

	// in case of a configured log level of debug or a message level of either error or fatal display the
	// source code file and line as well
	if Config.LogLevel == DebugLevel || level == ErrorLevel || level == FatalLevel {
		log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmsgprefix)
	} else {
		log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	}

	if len(string(level)) > 1 {
		log.SetPrefix(fmt.Sprintf("%s %s: ", appName, level))
	} else {
		log.SetPrefix(fmt.Sprintf("%s: ", appName))
	}
	if level == FatalLevel || (level == ErrorLevel && Config.LogLevel == DebugLevel) {
		msg = append(msg, string(debug.Stack()))
		log.Fatalf(format+"%s", msg...)
	}
	Logf(ErrorLevel, format, msg...)
}

// Logln is a wrapper for log.Println or log.Fatalln if the log level is set to FatalLevel.
// The log message is ignored when the level is below the configured log level from Config.LogLevel.
// If level is set to ErrorLevel or FatalLevel the file name and line of the source file is included and in case of the
// FatalLevel a stack trace added.
// If the configured log level in Config.LogLevel is set to DebugLevel the stack trace will be added to ErrorLevel entries
// as well.
func Logln(level Loglevel, msg ...any) {
	if level < Config.LogLevel {
		return
	}

	// in case of a configured log level of debug or a message level of either error or fatal display the
	// source code file and line as well
	if Config.LogLevel == DebugLevel || level == ErrorLevel || level == FatalLevel {
		log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmsgprefix)
	} else {
		log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	}

	// remove whitespace before : when level string is empty
	if len(string(level)) > 1 {
		log.SetPrefix(fmt.Sprintf("%s %s: ", appName, level))
	} else {
		log.SetPrefix(fmt.Sprintf("%s: ", appName))
	}

	// add stack trace to log message
	if level == FatalLevel || (level == ErrorLevel && Config.LogLevel == DebugLevel) {
		msg = append(msg, string(debug.Stack()))
	}

	// panic if loglevel is fatal
	if level == FatalLevel {
		log.Fatalln(msg...)
	}

	// output log message otherwise
	log.Println(msg...)
}
