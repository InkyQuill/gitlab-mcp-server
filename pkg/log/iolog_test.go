package log

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIOLogger(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	in := strings.NewReader("test input")
	out := &bytes.Buffer{}

	iol := NewIOLogger(in, out, logger)

	assert.NotNil(t, iol)
	assert.Same(t, in, iol.in)
	assert.Same(t, out, iol.out)
	assert.Same(t, logger, iol.logger)
}

func TestIOLogger_Read(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectError    bool
		expectLog      bool
		expectedLogMsg string
	}{
		{
			name:        "Read empty input",
			input:       "",
			expectError: true, // EOF expected
			expectLog:   false,
		},
		{
			name:           "Read some data",
			input:          "test data",
			expectError:    false,
			expectLog:      true,
			expectedLogMsg: "test data",
		},
		{
			name:           "Read JSON data",
			input:          `{"jsonrpc":"2.0","method":"test"}`,
			expectError:    false,
			expectLog:      true,
			expectedLogMsg: "jsonrpc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New()
			logger.SetLevel(log.DebugLevel)

			// Capture log output
			logBuf := &bytes.Buffer{}
			logger.SetOutput(logBuf)

			in := strings.NewReader(tc.input)
			out := &bytes.Buffer{}

			iol := NewIOLogger(in, out, logger)

			buf := make([]byte, 1024)
			n, err := iol.Read(buf)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if len(tc.input) > 0 {
					assert.Greater(t, n, 0)
				}
			}

			logOutput := logBuf.String()
			if tc.expectLog && len(tc.input) > 0 {
				assert.Contains(t, logOutput, "IN:")
				assert.Contains(t, logOutput, tc.expectedLogMsg)
			}
		})
	}
}

func TestIOLogger_Write(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectLog bool
	}{
		{
			name:      "Write empty data",
			input:     "",
			expectLog: true, // Logs even if empty
		},
		{
			name:      "Write simple data",
			input:     "output data",
			expectLog: true,
		},
		{
			name:      "Write JSON data",
			input:     `{"jsonrpc":"2.0","result":"success"}`,
			expectLog: true,
		},
		{
			name:      "Write JSON with token",
			input:     `{"token":"secret123","method":"test"}`,
			expectLog: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New()
			logger.SetLevel(log.DebugLevel)

			// Capture log output
			logBuf := &bytes.Buffer{}
			logger.SetOutput(logBuf)

			in := strings.NewReader("")
			out := &bytes.Buffer{}

			iol := NewIOLogger(in, out, logger)

			n, err := iol.Write([]byte(tc.input))

			require.NoError(t, err)
			assert.Equal(t, len(tc.input), n)

			// Verify output was written
			assert.Equal(t, tc.input, out.String())

			// Verify logging
			logOutput := logBuf.String()
			if tc.expectLog {
				assert.Contains(t, logOutput, "OUT:")
			}
		})
	}
}

func TestRedactSensitive(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectRedacted  bool
		redactedFields  []string // fields that should be redacted
		expectUnchanged bool
	}{
		{
			name:            "Non-JSON input",
			input:           "plain text message",
			expectUnchanged: true,
		},
		{
			name:            "JSON without sensitive fields",
			input:           `{"method":"test","params":{"arg":"value"}}`,
			expectUnchanged: true,
		},
		{
			name:           "JSON with token field at root",
			input:          `{"token":"secret123","method":"test"}`,
			expectRedacted: true,
			redactedFields: []string{"token"},
		},
		{
			name:           "JSON with password field",
			input:          `{"password":"mypass","user":"admin"}`,
			expectRedacted: true,
			redactedFields: []string{"password"},
		},
		{
			name:           "JSON with secret field",
			input:          `{"secret":"mysecret","action":"authenticate"}`,
			expectRedacted: true,
			redactedFields: []string{"secret"},
		},
		{
			name:           "JSON with apiKey field",
			input:          `{"apiKey":"key123","resource":"data"}`,
			expectRedacted: true,
			redactedFields: []string{"apiKey"},
		},
		{
			name:           "JSON with accessToken field",
			input:          `{"accessToken":"token123","scope":"read"}`,
			expectRedacted: true,
			redactedFields: []string{"accessToken"},
		},
		{
			name:           "JSON with multiple sensitive fields",
			input:          `{"token":"t1","password":"p1","user":"u1"}`,
			expectRedacted: true,
			redactedFields: []string{"token", "password"},
		},
		{
			name:           "JSON with sensitive field in params",
			input:          `{"method":"auth","params":{"token":"secret123"}}`,
			expectRedacted: true,
			redactedFields: []string{"token"},
		},
		{
			name:           "JSON with sensitive in both root and params",
			input:          `{"token":"t1","params":{"password":"p1"}}`,
			expectRedacted: true,
			redactedFields: []string{"token", "password"},
		},
		{
			name:            "Invalid JSON",
			input:           `{invalid json}`,
			expectUnchanged: true,
		},
		{
			name:            "Empty JSON object",
			input:           `{}`,
			expectUnchanged: true,
		},
		{
			name:            "JSON array - not processed (only objects)",
			input:           `[{"token":"secret"}]`,
			expectUnchanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := redactSensitive(tc.input)

			if tc.expectUnchanged {
				assert.Equal(t, tc.input, result)
			} else if tc.expectRedacted {
				// Parse result as JSON to check redaction
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(result), &parsed)
				require.NoError(t, err, "Result should be valid JSON")

				// Check redacted fields at root level
				for _, field := range tc.redactedFields {
					if val, ok := parsed[field]; ok {
						assert.Equal(t, "***REDACTED***", val, "Field %s should be redacted", field)
					}
				}

				// Check params nested level
				if params, ok := parsed["params"].(map[string]interface{}); ok {
					for _, field := range tc.redactedFields {
						if val, ok := params[field]; ok {
							assert.Equal(t, "***REDACTED***", val, "Field %s in params should be redacted", field)
						}
					}
				}
			}
		})
	}
}

func TestRedactSensitive_Truncation(t *testing.T) {
	// Create a large JSON message (over 2000 chars)
	largeValue := strings.Repeat("a", 3000)
	input := `{"data":"` + largeValue + `"}`

	result := redactSensitive(input)

	// Should be truncated
	assert.Less(t, len(result), 3000)
	assert.Contains(t, result, "... (truncated)")
}

func TestIOLogger_ReadWriteIntegration(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	// Create pipes for bidirectional communication
	inReader, _ := io.Pipe()
	outReader, outWriter := io.Pipe()

	iol := NewIOLogger(inReader, outWriter, logger)

	// Test writing through the logger
	go func() {
		iol.Write([]byte(`{"method":"test","token":"secret"}`))
	}()

	// Read from the output pipe
	buf := make([]byte, 1024)
	n, err := outReader.Read(buf)
	require.NoError(t, err)
	assert.Greater(t, n, 0)

	// Verify the data was written (not redacted in actual output)
	output := string(buf[:n])
	assert.Contains(t, output, `"method":"test"`)
	assert.Contains(t, output, `"token":"secret"`)
}

func TestIsLikelyJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "JSON object",
			input:    `{"key":"value"}`,
			expected: true,
		},
		{
			name:     "JSON array",
			input:    `[1,2,3]`,
			expected: true,
		},
		{
			name:     "JSON with leading whitespace",
			input:    `  {"key":"value"}`,
			expected: true,
		},
		{
			name:     "JSON with leading newline",
			input:    "\n{\"key\":\"value\"}",
			expected: true,
		},
		{
			name:     "Not JSON - plain text",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "Not JSON - XML",
			input:    `<xml>tag</xml>`,
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Not JSON - number",
			input:    "123",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isLikelyJSON(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
