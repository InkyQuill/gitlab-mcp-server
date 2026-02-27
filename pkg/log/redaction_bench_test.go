package log

import (
	"testing"
)

func BenchmarkRedactJSON_Small(b *testing.B) {
	input := `{"token":"glpat-12345678901234567890"}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkRedactJSON_Large(b *testing.B) {
	input := `{"params":{"arguments":{"token":"glpat-12345678901234567890","user":"test"}}}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkRedactJSON_MultipleFields(b *testing.B) {
	input := `{"token":"glpat-12345678901234567890","password":"secret123","apiKey":"key-123","accessToken":"token-456"}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkRedactJSON_WithParams(b *testing.B) {
	input := `{"method":"tools/call","params":{"name":"getIssue","arguments":{"token":"glpat-12345678901234567890","projectId":"123"}}}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkRedactJSON_NoSensitiveData(b *testing.B) {
	input := `{"method":"tools/call","params":{"name":"getIssue","arguments":{"projectId":"123","issueIid":1}}}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkRedactJSON_InvalidJSON(b *testing.B) {
	input := `this is not json`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkRedactJSON_TruncationNeeded(b *testing.B) {
	// Create a large JSON payload that exceeds 2000 chars
	input := `{"method":"tools/call","params":{"name":"getIssue","arguments":{"projectId":"123","issueIid":1,"token":"glpat-12345678901234567890","data":"` +
		string(make([]byte, 2500)) + `"}}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = redactSensitive(input)
	}
}

func BenchmarkIsLikelyJSON_True(b *testing.B) {
	input := `{"key":"value"}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = isLikelyJSON(input)
	}
}

func BenchmarkIsLikelyJSON_False(b *testing.B) {
	input := `plain text`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = isLikelyJSON(input)
	}
}
