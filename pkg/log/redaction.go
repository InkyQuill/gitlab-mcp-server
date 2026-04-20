package log

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Patterns for redacting sensitive tokens
var (
	// GitLab tokens: glpat-xxxxxxxxxxxx
	gitlabTokenPattern = regexp.MustCompile(`glpat-[a-zA-Z0-9_-]{20,}`)
	// GitLab personal access tokens (older format): private_token-xxxxxxxxxxxx
	gitlabPrivateTokenPattern = regexp.MustCompile(`private[_-]?token\s*[:=]\s*[a-zA-Z0-9_-]{20,}`)
	// GitHub tokens: ghp_xxxxxxxxxxxxxxxxxx, gho_xxxxxxxxxxxxxxxxxx, ghu_xxxxxxxxxxxxxxxxxx, ghs_xxxxxxxxxxxxxxxxxx
	githubTokenPattern = regexp.MustCompile(`gh[opsu]_[a-zA-Z0-9]{36}`)
	// GitHub classic tokens (still sometimes used):ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
	githubClassicTokenPattern = regexp.MustCompile(`gh[po]_[a-zA-Z0-9_-]{20,}`)
	// JWT tokens
	jwtPattern = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)
	// Bearer tokens
	bearerPattern = regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-._~+/+=]+`)
	// API keys in URL query parameters
	apiKeyInURL = regexp.MustCompile(`([?&])(api[_-]?key|token|auth|access[_-]?token|private[_-]?token|secret|apikey)[=][^&\s]+`)
	// Generic token pattern - matches long alphanumeric strings that look like tokens
	// Only applies in non-JSON contexts (after JSON parsing is attempted)
	//
	//nolint:unused // kept for future use; opt-in redaction pass will wire it up
	genericTokenPattern = regexp.MustCompile(`\b[a-zA-Z0-9_-]{32,}\b`)
	// Basic auth in URLs - matches //user:password@ where password can contain anything but @
	// Uses a non-greedy match to handle passwords with special characters
	basicAuthPattern = regexp.MustCompile(`//[^:@]+:(?:[^@]+?)?@`)
	// Email addresses (often sensitive in logs)
	//
	//nolint:unused // kept for future use; opt-in redaction pass will wire it up
	emailPattern = regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`)
)

// Redactor recursively removes sensitive data from objects
type Redactor struct {
	sensitiveKeys map[string]bool
	maxLength     int // Maximum length for redacted strings
}

// NewRedactor creates a new redactor with default sensitive keys
func NewRedactor() *Redactor {
	return &Redactor{
		sensitiveKeys: map[string]bool{
			"token":         true,
			"accesstoken":   true,
			"access_token":  true,
			"apikey":        true,
			"api_key":       true,
			"secret":        true,
			"password":      true,
			"passwd":        true,
			"privatekey":    true,
			"private_key":   true,
			"private-token": true,
			"authorization": true,
			"bearer":        true,
			"gitlab_token":  true,
			"github_token":  true,
			"auth":          true,
			"credentials":   true,
			"sessiontoken":  true,
			"session_token": true,
			"refreshtoken":  true,
			"refresh_token": true,
		},
		maxLength: 2000, // Default max length for redacted output
	}
}

// WithMaxLength sets the maximum length for redacted output
func (r *Redactor) WithMaxLength(max int) *Redactor {
	r.maxLength = max
	return r
}

// WithSensitiveKeys adds custom keys to the sensitive keys list
func (r *Redactor) WithSensitiveKeys(keys ...string) *Redactor {
	for _, key := range keys {
		lowerKey := strings.ToLower(key)
		r.sensitiveKeys[lowerKey] = true
	}
	return r
}

// RedactJSON removes sensitive data from JSON string
func (r *Redactor) RedactJSON(input string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		// If not JSON, apply patterns to the string
		return r.truncate(r.redactString(input))
	}

	// Recursively redact
	data = r.redactValue(data)

	// Back to JSON
	result, _ := json.Marshal(data)
	return r.truncate(string(result))
}

// redactValue recursively processes a value
func (r *Redactor) redactValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return r.redactMap(val)
	case []interface{}:
		return r.redactArray(val)
	case string:
		return r.redactString(val)
	case float64, bool, nil:
		return v
	default:
		// For unknown types, convert to string and redact
		return r.redactString(fmt.Sprintf("%v", val))
	}
}

// redactMap processes a JSON object
func (r *Redactor) redactMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range m {
		lowerKey := strings.ToLower(key)
		// Check both exact match and contains match for sensitive keys
		isSensitive := r.sensitiveKeys[lowerKey]
		if !isSensitive {
			// Check if any sensitive key is contained in the current key
			for sensitiveKey := range r.sensitiveKeys {
				if strings.Contains(lowerKey, sensitiveKey) {
					isSensitive = true
					break
				}
			}
		}

		if isSensitive {
			result[key] = "***REDACTED***"
		} else {
			result[key] = r.redactValue(value)
		}
	}
	return result
}

// redactArray processes a JSON array
func (r *Redactor) redactArray(arr []interface{}) []interface{} {
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		result[i] = r.redactValue(v)
	}
	return result
}

// redactString processes a string value
func (r *Redactor) redactString(s string) string {
	// Apply all patterns in order
	result := gitlabTokenPattern.ReplaceAllString(s, "glpat-***REDACTED***")
	result = gitlabPrivateTokenPattern.ReplaceAllString(result, "private_token-***REDACTED***")
	result = githubTokenPattern.ReplaceAllString(result, "***GITHUB_TOKEN_REDACTED***")
	result = githubClassicTokenPattern.ReplaceAllString(result, "***GITHUB_TOKEN_REDACTED***")
	result = jwtPattern.ReplaceAllString(result, "***JWT_REDACTED***")
	result = bearerPattern.ReplaceAllString(result, "Bearer ***REDACTED***")
	result = apiKeyInURL.ReplaceAllString(result, "$1***REDACTED***")
	result = basicAuthPattern.ReplaceAllString(result, "//***REDACTED***@")

	return result
}

// truncate truncates string to max length if needed
func (r *Redactor) truncate(s string) string {
	if r.maxLength > 0 && len(s) > r.maxLength {
		return s[:r.maxLength] + "... (truncated)"
	}
	return s
}

// RedactString removes sensitive data from a string (non-JSON)
func (r *Redactor) RedactString(input string) string {
	return r.truncate(r.redactString(input))
}

// RedactString is a convenience function for string redaction with default redactor
func RedactString(input string) string {
	r := NewRedactor()
	return r.RedactString(input)
}

// RedactJSON is a convenience function for JSON redaction with default redactor
func RedactJSON(input string) string {
	r := NewRedactor()
	return r.RedactJSON(input)
}
