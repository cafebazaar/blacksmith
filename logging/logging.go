package logging

import "fmt"

type logEntry struct {
	Subsystem string
	Debug     bool
	Msg       string
}

var logCh = make(chan logEntry)

type minimalLogger interface {
	Printf(format string, v ...interface{})
}

// RecordLogs prints logs into the logger according to thier level and requested
// debugging level (debug=true for more logs)
func RecordLogs(logger minimalLogger, debug bool) {
	for l := range logCh {
		if l.Debug && !debug {
			continue
		}
		level := 'I'
		if l.Debug {
			level = 'D'
		}
		logger.Printf("[%s]%c %s", l.Subsystem, level, l.Msg)
	}
}

// Log wrtites Info level logs to logger
func Log(subsystem string, msg string, args ...interface{}) {
	logCh <- logEntry{subsystem, false, fmt.Sprintf(msg, args...)}
}

// Debug wrtites Debug level logs to logger
func Debug(subsystem string, msg string, args ...interface{}) {
	logCh <- logEntry{subsystem, true, fmt.Sprintf(msg, args...)}
}
