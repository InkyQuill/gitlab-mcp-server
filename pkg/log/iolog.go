package log

import (
	"encoding/json"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
)

// IOLogger wraps io.Reader/io.Writer to log JSON-RPC messages
type IOLogger struct {
	in     io.Reader
	out    io.Writer
	logger *log.Logger
}

// NewIOLogger creates a new IOLogger instance
func NewIOLogger(in io.Reader, out io.Writer, logger *log.Logger) *IOLogger {
	return &IOLogger{
		in:     in,
		out:    out,
		logger: logger,
	}
}

// Read implements io.Reader, logging incoming messages
func (iol *IOLogger) Read(p []byte) (n int, err error) {
	n, err = iol.in.Read(p)
	if n > 0 {
		logged := redactSensitive(string(p[:n]))
		iol.logger.Debugf("IN: %s", logged)
	}
	return
}

// Write implements io.Writer, logging outgoing messages
func (iol *IOLogger) Write(p []byte) (n int, err error) {
	logged := redactSensitive(string(p))
	iol.logger.Debugf("OUT: %s", logged)
	return iol.out.Write(p)
}

// redactSensitive removes sensitive fields from JSON-RPC messages
func redactSensitive(msg string) string {
	// Parse as JSON
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(msg), &raw); err != nil {
		return msg // Not JSON, return as-is
	}

	// Redact common sensitive fields
	sensitiveKeys := []string{"token", "password", "secret", "apiKey", "accessToken"}
	for _, key := range sensitiveKeys {
		if _, ok := raw[key]; ok {
			raw[key] = "***REDACTED***"
		}
	}

	// Handle nested "params" object
	if params, ok := raw["params"].(map[string]interface{}); ok {
		for _, key := range sensitiveKeys {
			if _, ok := params[key]; ok {
				params[key] = "***REDACTED***"
			}
		}
	}

	// Marshal back to JSON with compact formatting
	redacted, _ := json.Marshal(raw)
	result := string(redacted)

	// Truncate if too long (limit to 2000 chars for readability)
	if len(result) > 2000 {
		result = result[:2000] + "... (truncated)"
	}

	return result
}

// isLikelyJSON checks if a string appears to be JSON
func isLikelyJSON(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}
