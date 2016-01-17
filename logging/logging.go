package logging // import "github.com/cafebazaar/blacksmith/logging"

import (
	"fmt"
	"net/http"
)

const logFormat = "%c[%s] %s"

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
		logger.Printf(logFormat, level, l.Subsystem, l.Msg)
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

func httpRequestMsg(r *http.Request) string {
	return fmt.Sprintf("%s RemoteAddr:%s Method:%s Referer:%s UserAgent:%s", r.URL, r.RemoteAddr, r.Method, r.Referer(), r.UserAgent())
}

// LogHTTPRequest wrtites Info level logs of HTTP requests to logger
func LogHTTPRequest(subsystem string, r *http.Request) {
	logCh <- logEntry{subsystem, false, httpRequestMsg(r)}
}

// DebugHTTPRequest wrtites Debug level logs of HTTP requests to logger
func DebugHTTPRequest(subsystem string, r *http.Request) {
	logCh <- logEntry{subsystem, true, httpRequestMsg(r)}
}
