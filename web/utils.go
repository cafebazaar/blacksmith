package web

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/cafebazaar/blacksmith/logging"
)
const (
	debugTag = "WEB"
)

// Generates a random string with the given length by reading /dev/urandom and encoding it to
func urandomString(n int) string {
	random_bytes := make([]byte, n)
	_, err := rand.Read(random_bytes)
	if err != nil {
		logging.Debug(debugTag, "could'nt generate random string due to: %s", err)
	}
	// the base64 encoded string length is 4/3 times longer, so we just need the first n characters
	return base64.StdEncoding.EncodeToString(random_bytes)[:n]
}