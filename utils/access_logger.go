package utils

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// LogAccess constructs a log.Entry from the given http.Request
func LogAccess(r *http.Request) *log.Entry {
	return log.WithFields(log.Fields{
		"host":       r.Host,
		"method":     r.Method,
		"uri":        r.RequestURI,
		"proto":      r.Proto,
		"referer":    r.Referer(),
		"user-agent": r.UserAgent(),
	})
}
