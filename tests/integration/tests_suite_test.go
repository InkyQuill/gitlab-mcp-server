package integration

import (
	"os"
	"testing"
)

// TestMain is the entry point for all integration tests
func TestMain(m *testing.M) {
	// Setup for all integration tests can be done here

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}
