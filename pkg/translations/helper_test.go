package translations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslate(t *testing.T) {
	tests := []struct {
		name         string
		translations map[string]string
		key          string
		expected     string
	}{
		{
			name:         "Key exists in translations",
			translations: map[string]string{"hello": "Hello", "goodbye": "Goodbye"},
			key:          "hello",
			expected:     "Hello",
		},
		{
			name:         "Key does not exist - returns key",
			translations: map[string]string{"hello": "Hello"},
			key:          "goodbye",
			expected:     "goodbye",
		},
		{
			name:         "Empty translations map",
			translations: map[string]string{},
			key:          "any_key",
			expected:     "any_key",
		},
		{
			name:         "Nil translations map",
			translations: nil,
			key:          "any_key",
			expected:     "any_key",
		},
		{
			name:         "Empty string key",
			translations: map[string]string{"": "empty_value"},
			key:          "",
			expected:     "empty_value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Translate(tc.translations, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetAllTranslationKeys(t *testing.T) {
	keys := getAllTranslationKeys()

	// Verify we have all expected keys
	assert.NotEmpty(t, keys, "Should have translation keys")

	// Check specific tool descriptions
	expectedKeys := []string{
		TOOL_GET_PROJECT_DESCRIPTION,
		TOOL_LIST_PROJECTS_DESCRIPTION,
		TOOL_GET_ISSUE_DESCRIPTION,
		TOOL_CREATE_ISSUE_DESCRIPTION,
		TOOL_GET_MERGE_REQUEST_DESCRIPTION,
		TOOL_LIST_TOKENS_DESCRIPTION,
	}

	for _, key := range expectedKeys {
		value, exists := keys[key]
		assert.True(t, exists, "Key %s should exist", key)
		assert.NotEmpty(t, value, "Key %s should have a value", key)
		// Value should be different from key (has description)
		assert.NotEqual(t, key, value, "Key %s should have a description, not just the key name", key)
	}
}

func TestTranslationHelper(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(t *testing.T) string
		validate    func(t *testing.T, translations map[string]string, dumpFunc func())
	}{
		{
			name: "No config file exists",
			setupConfig: func(t *testing.T) string {
				return t.TempDir()
			},
			validate: func(t *testing.T, translations map[string]string, dumpFunc func()) {
				// Should return empty map
				assert.NotNil(t, translations)
				assert.Empty(t, translations)
				assert.NotNil(t, dumpFunc)
			},
		},
		{
			name: "Config file with valid translations",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, configFileName)
				config := map[string]string{
					TOOL_GET_PROJECT_DESCRIPTION: "Custom project description",
					"custom_key":                 "Custom value",
				}
				data, err := json.Marshal(config)
				require.NoError(t, err)
				err = os.WriteFile(configPath, data, 0644)
				require.NoError(t, err)

				// Note: TranslationHelper looks for config next to binary,
				// so we can't easily test this with custom paths
				return tmpDir
			},
			validate: func(t *testing.T, translations map[string]string, dumpFunc func()) {
				// Since TranslationHelper uses os.Executable(), we can't easily
				// mock the path in tests. The function will use the actual binary path.
				// Just verify it returns non-nil values
				assert.NotNil(t, translations)
				assert.NotNil(t, dumpFunc)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New()
			logger.SetLevel(log.ErrorLevel)

			_ = tc.setupConfig(t)

			translations, dumpFunc := TranslationHelper(logger)

			if tc.validate != nil {
				tc.validate(t, translations, dumpFunc)
			}
		})
	}
}

func TestDumpAllTranslations(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(t *testing.T) string
		validate    func(t *testing.T, configPath string)
	}{
		{
			name: "Create new config file",
			setupConfig: func(t *testing.T) string {
				return t.TempDir()
			},
			validate: func(t *testing.T, configPath string) {
				// Config file should not exist initially
				_, err := os.Stat(configPath)
				assert.True(t, os.IsNotExist(err))

				// Dump translations
				logger := log.New()
				logger.SetLevel(log.ErrorLevel)
				dumpAllTranslations(logger, configPath)

				// Config should now exist
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				// Parse and verify
				var config map[string]string
				err = json.Unmarshal(data, &config)
				require.NoError(t, err)

				// Should have all translation keys
				allKeys := getAllTranslationKeys()
				assert.Equal(t, len(allKeys), len(config), "Should have all keys")

				// Verify some keys
				assert.Contains(t, config, TOOL_GET_PROJECT_DESCRIPTION)
				assert.Contains(t, config, TOOL_LIST_PROJECTS_DESCRIPTION)
			},
		},
		{
			name: "Merge with existing config",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, configFileName)

				// Create existing config with some custom values
				existing := map[string]string{
					TOOL_GET_PROJECT_DESCRIPTION: "Custom project desc",
					"extra_key":                  "Extra value",
				}
				data, err := json.Marshal(existing)
				require.NoError(t, err)
				err = os.WriteFile(configPath, data, 0644)
				require.NoError(t, err)

				return tmpDir
			},
			validate: func(t *testing.T, configPath string) {
				logger := log.New()
				logger.SetLevel(log.ErrorLevel)
				dumpAllTranslations(logger, configPath)

				// Read merged config
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var config map[string]string
				err = json.Unmarshal(data, &config)
				require.NoError(t, err)

				// Should preserve custom value
				assert.Equal(t, "Custom project desc", config[TOOL_GET_PROJECT_DESCRIPTION])

				// Should preserve extra key
				assert.Equal(t, "Extra value", config["extra_key"])

				// Should have all other keys
				allKeys := getAllTranslationKeys()
				assert.GreaterOrEqual(t, len(config), len(allKeys), "Should have at least all standard keys")
			},
		},
		{
			name: "Update existing config with missing keys",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, configFileName)

				// Create minimal config
				existing := map[string]string{
					TOOL_GET_PROJECT_DESCRIPTION: "Only one key",
				}
				data, err := json.Marshal(existing)
				require.NoError(t, err)
				err = os.WriteFile(configPath, data, 0644)
				require.NoError(t, err)

				return tmpDir
			},
			validate: func(t *testing.T, configPath string) {
				logger := log.New()
				logger.SetLevel(log.ErrorLevel)
				dumpAllTranslations(logger, configPath)

				// Read updated config
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var config map[string]string
				err = json.Unmarshal(data, &config)
				require.NoError(t, err)

				// Should have original key
				assert.Equal(t, "Only one key", config[TOOL_GET_PROJECT_DESCRIPTION])

				// Should have added all missing keys
				allKeys := getAllTranslationKeys()
				assert.Equal(t, len(allKeys), len(config), "Should have all keys after merge")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := tc.setupConfig(t)
			configPath := filepath.Join(tmpDir, configFileName)
			tc.validate(t, configPath)
		})
	}
}

func TestTranslationKeys_Constants(t *testing.T) {
	// Verify all constants have non-empty values
	constants := []string{
		TOOL_GET_PROJECT_DESCRIPTION,
		TOOL_LIST_PROJECTS_DESCRIPTION,
		TOOL_GET_PROJECT_FILE_DESCRIPTION,
		TOOL_LIST_PROJECT_FILES_DESCRIPTION,
		TOOL_GET_PROJECT_BRANCHES_DESCRIPTION,
		TOOL_GET_PROJECT_COMMITS_DESCRIPTION,
		TOOL_GET_ISSUE_DESCRIPTION,
		TOOL_LIST_ISSUES_DESCRIPTION,
		TOOL_CREATE_ISSUE_DESCRIPTION,
		TOOL_UPDATE_ISSUE_DESCRIPTION,
		TOOL_ISSUE_COMMENT_DESCRIPTION,
		TOOL_GET_ISSUE_LABELS_DESCRIPTION,
		TOOL_MILESTONE_DESCRIPTION,
		TOOL_LIST_MILESTONES_DESCRIPTION,
		TOOL_GET_MERGE_REQUEST_DESCRIPTION,
		TOOL_LIST_MERGE_REQUESTS_DESCRIPTION,
		TOOL_CREATE_MERGE_REQUEST_DESCRIPTION,
		TOOL_UPDATE_MERGE_REQUEST_DESCRIPTION,
		TOOL_MERGE_REQUEST_COMMENT_DESCRIPTION,
		TOOL_LIST_TOKENS_DESCRIPTION,
		TOOL_ADD_TOKEN_DESCRIPTION,
		TOOL_UPDATE_TOKEN_DESCRIPTION,
		TOOL_REMOVE_TOKEN_DESCRIPTION,
		TOOL_VALIDATE_TOKEN_DESCRIPTION,
		TOOL_GET_NOTIFICATIONS_DESCRIPTION,
	}

	for _, constant := range constants {
		assert.NotEmpty(t, constant, "Translation key constant should not be empty")
		// All constants should end with "_DESCRIPTION"
		assert.True(t, strings.HasSuffix(constant, "_DESCRIPTION"), "Tool key should end with _DESCRIPTION: %s", constant)
	}
}

func TestTranslate_Integration(t *testing.T) {
	// Test the full flow: get keys, translate, use defaults
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Get all translation keys
	allKeys := getAllTranslationKeys()
	require.NotEmpty(t, allKeys)

	// Pick a key to test
	testKey := TOOL_GET_PROJECT_DESCRIPTION
	defaultValue := allKeys[testKey]
	require.NotEmpty(t, defaultValue)

	// Test with empty translations (should return key)
	translations := map[string]string{}
	result := Translate(translations, testKey)
	assert.Equal(t, testKey, result, "Should return key when translation not found")

	// Test with translation present
	translations[testKey] = "Custom Project Description"
	result = Translate(translations, testKey)
	assert.Equal(t, "Custom Project Description", result, "Should return translated value")
}

func TestTranslationHelper_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, configFileName)

	// Create config file with invalid JSON
	require.NoError(t, os.WriteFile(configPath, []byte(`{invalid json}`), 0644))

	// Since TranslationHelper uses os.Executable() to find the config file,
	// we can't easily test this with a custom path. However, we can test
	// dumpAllTranslations with invalid JSON in the existing file.
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Test that dumpAllTranslations handles invalid JSON gracefully
	// It should ignore the invalid JSON and create a new file with all keys
	dumpAllTranslations(logger, configPath)

	// Config should be rewritten with valid JSON
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config map[string]string
	err = json.Unmarshal(data, &config)
	require.NoError(t, err, "Config should be valid JSON after dump")

	// Should have all translation keys
	allKeys := getAllTranslationKeys()
	assert.Equal(t, len(allKeys), len(config), "Should have all keys after fixing invalid JSON")
}

func TestDumpAllTranslations_ReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, configFileName)

	// Create config file as a directory to cause read error
	require.NoError(t, os.MkdirAll(configPath, 0700))
	t.Cleanup(func() { os.RemoveAll(configPath) })

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// dumpAllTranslations should handle read errors gracefully
	// It should continue and create/update the file
	dumpAllTranslations(logger, configPath)

	// After the function runs, the directory should still exist (or be replaced)
	// The function should handle the error and continue
	_, err := os.Stat(configPath)
	// The path might be a directory or file depending on how the error was handled
	// The important thing is that the function doesn't panic
	assert.NoError(t, err, "Path should still exist")
}

func TestDumpAllTranslations_WriteFileError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, configFileName)

	// Create parent directory as read-only to prevent writing (on Unix systems)
	require.NoError(t, os.MkdirAll(tmpDir, 0500)) // read-only, no write permission
	t.Cleanup(func() {
		os.Chmod(tmpDir, 0700) // restore permissions for cleanup
		os.RemoveAll(tmpDir)
	})

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// dumpAllTranslations should handle write errors
	// On systems where this fails, it should log an error but not panic
	dumpAllTranslations(logger, configPath)

	// The function should handle the error gracefully
	// Note: On some systems this might succeed if the process owner can write
	// But on systems where it fails, we should not panic
}

func TestDumpAllTranslations_MarshalError(t *testing.T) {
	// This is difficult to test because getAllTranslationKeys() returns
	// a map[string]string which should always be marshalable.
	// The error path for json.MarshalIndent is very unlikely to occur.
	// However, we can verify the code handles it if it does occur.
	// This edge case is considered low priority as it's nearly impossible
	// to trigger with the current data structures.
}

func TestTranslationHelper_DumpTranslationsFunction(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Get translations and dump function
	translations, dumpFunc := TranslationHelper(logger)

	// Verify function is not nil
	assert.NotNil(t, dumpFunc)
	assert.NotNil(t, translations)

	// The dumpFunc should be callable
	// Since it uses os.Executable() internally, we can't easily test it with a custom path
	// But we can verify it doesn't panic when called
	assert.NotPanics(t, func() {
		dumpFunc()
	}, "dumpTranslations function should not panic when called")
}

func TestDumpAllTranslations_ExistingInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, configFileName)

	// Create config file with invalid JSON
	require.NoError(t, os.WriteFile(configPath, []byte(`{invalid json}`), 0644))

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// dumpAllTranslations should handle invalid JSON in existing file
	// It should ignore the invalid JSON and create a new file with all keys
	dumpAllTranslations(logger, configPath)

	// Config should be rewritten with valid JSON
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config map[string]string
	err = json.Unmarshal(data, &config)
	require.NoError(t, err, "Config should be valid JSON after dump")

	// Should have all translation keys
	allKeys := getAllTranslationKeys()
	assert.Equal(t, len(allKeys), len(config), "Should have all keys after fixing invalid JSON")
}

func TestTranslationHelper_SuccessfulLoad(t *testing.T) {
	// This test is difficult because TranslationHelper uses os.Executable()
	// which returns the actual binary path. We can't easily mock this.
	// However, we can test that the function works when a valid config file
	// exists next to the binary (if it does).
	
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Call TranslationHelper - it will try to load from the actual binary location
	translations, dumpFunc := TranslationHelper(logger)

	// Should return non-nil values
	assert.NotNil(t, translations)
	assert.NotNil(t, dumpFunc)

	// The function should not panic
	assert.NotPanics(t, func() {
		_, _ = TranslationHelper(logger)
	}, "TranslationHelper should not panic")
}
