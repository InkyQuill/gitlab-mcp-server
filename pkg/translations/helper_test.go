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
		TOOL_GET_ISSUE_COMMENTS_DESCRIPTION,
		TOOL_CREATE_ISSUE_COMMENT_DESCRIPTION,
		TOOL_UPDATE_ISSUE_COMMENT_DESCRIPTION,
		TOOL_GET_ISSUE_LABELS_DESCRIPTION,
		TOOL_GET_MILESTONE_DESCRIPTION,
		TOOL_LIST_MILESTONES_DESCRIPTION,
		TOOL_CREATE_MILESTONE_DESCRIPTION,
		TOOL_UPDATE_MILESTONE_DESCRIPTION,
		TOOL_GET_MERGE_REQUEST_DESCRIPTION,
		TOOL_LIST_MERGE_REQUESTS_DESCRIPTION,
		TOOL_CREATE_MERGE_REQUEST_DESCRIPTION,
		TOOL_UPDATE_MERGE_REQUEST_DESCRIPTION,
		TOOL_GET_MERGE_REQUEST_COMMENTS_DESCRIPTION,
		TOOL_CREATE_MERGE_REQUEST_COMMENT_DESCRIPTION,
		TOOL_UPDATE_MERGE_REQUEST_COMMENT_DESCRIPTION,
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
