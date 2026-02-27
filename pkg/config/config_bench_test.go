package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func BenchmarkManagerSave(b *testing.B) {
	// Create a temp directory for testing
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	m, err := NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	// Setup test config
	m.config.Servers = map[string]*ServerConfig{
		"test": {Name: "test", Host: "https://gitlab.com", Token: "test-token"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = m.Save()
	}
}

func BenchmarkManagerLoad(b *testing.B) {
	// Create a temp directory for testing
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	// Create initial config
	m, err := NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}
	m.config.Servers = map[string]*ServerConfig{
		"test": {Name: "test", Host: "https://gitlab.com", Token: "test-token"},
	}
	if err := m.Save(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m2, _ := NewManager(configPath)
		_ = m2.Load()
	}
}

func BenchmarkManagerAddServer(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	m, err := NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	server := &ServerConfig{
		Name:  "test",
		Host:  "https://gitlab.com",
		Token: "test-token",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Use unique names for each iteration
		server.Name = "test"
		m.config.Servers = make(map[string]*ServerConfig) // Reset map
		_ = m.AddServer(server)
	}
}

func BenchmarkManagerGetServer(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	m, err := NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}
	m.config.Servers = map[string]*ServerConfig{
		"test": {Name: "test", Host: "https://gitlab.com", Token: "test-token"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = m.GetServer("test")
	}
}

func BenchmarkManagerListServers(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	m, err := NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	// Add multiple servers
	for i := 0; i < 10; i++ {
		name := "server" + string(rune('0'+i))
		m.config.Servers[name] = &ServerConfig{
			Name:  name,
			Host:  "https://gitlab.com",
			Token: "test-token",
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = m.ListServers()
	}
}

func BenchmarkMarshalConfig(b *testing.B) {
	cfg := &Config{
		Version: "1.0",
		Servers: map[string]*ServerConfig{
			"test1": {Name: "test1", Host: "https://gitlab.example.com", Token: "glpat-12345678901234567890", UserID: 123, Username: "testuser"},
			"test2": {Name: "test2", Host: "https://gitlab.com", Token: "glpat-09876543210987654321", UserID: 456, Username: "anotheruser"},
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(cfg)
	}
}

func BenchmarkUnmarshalConfig(b *testing.B) {
	configJSON := `{
		"version": "1.0",
		"servers": {
			"test1": {
				"name": "test1",
				"host": "https://gitlab.example.com",
				"token": "glpat-12345678901234567890",
				"userId": 123,
				"username": "testuser"
			},
			"test2": {
				"name": "test2",
				"host": "https://gitlab.com",
				"token": "glpat-09876543210987654321",
				"userId": 456,
				"username": "anotheruser"
			}
		}
	}`

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var cfg Config
		_ = json.Unmarshal([]byte(configJSON), &cfg)
	}
}
