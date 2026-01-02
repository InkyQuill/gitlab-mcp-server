package translations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	configFileName = "gitlab-mcp-server-config.json"
)

// TranslationHelper loads translations from config file
func TranslationHelper(logger *log.Logger) (map[string]string, func()) {
	translations := make(map[string]string)

	// Find config file next to binary
	execPath, err := os.Executable()
	if err != nil {
		logger.Debugf("Could not locate binary path for translations: %v", err)
		return translations, func() {}
	}

	configPath := filepath.Join(filepath.Dir(execPath), configFileName)

	// Load config if exists
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &translations); err != nil {
			logger.Warnf("Failed to parse translation config: %v", err)
		} else {
			logger.Infof("Loaded %d translations from %s", len(translations), configPath)
		}
	}

	// Return function to dump translations
	dumpTranslations = func() {
		dumpAllTranslations(logger, configPath)
	}

	return translations, dumpTranslations
}

// Translate returns translated string or key if not found
func Translate(translations map[string]string, key string) string {
	if translated, ok := translations[key]; ok {
		return translated
	}
	return key
}

// dumpTranslations is assigned by TranslationHelper
var dumpTranslations func()

// dumpAllTranslations generates template with all translation keys
func dumpAllTranslations(logger *log.Logger, configPath string) {
	// Collect all keys
	allKeys := getAllTranslationKeys()

	// Merge with existing config
	existing := make(map[string]string)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	// Add any missing keys
	for key, value := range allKeys {
		if _, exists := existing[key]; !exists {
			existing[key] = value
		}
	}

	// Write back with pretty formatting
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		logger.Errorf("Failed to marshal translations: %v", err)
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logger.Errorf("Failed to write translations: %v", err)
	} else {
		fmt.Fprintf(os.Stderr, "Exported %d translation keys to %s\n", len(existing), configPath)
		logger.Infof("Exported %d translation keys to %s", len(existing), configPath)
	}
}

// getAllTranslationKeys returns all translation keys with their default English values
func getAllTranslationKeys() map[string]string {
	return map[string]string{
		// Projects toolset
		TOOL_GET_PROJECT_DESCRIPTION:          "Retrieves details for a specific GitLab project.",
		TOOL_LIST_PROJECTS_DESCRIPTION:        "Lists GitLab projects, with optional filtering.",
		TOOL_GET_PROJECT_FILE_DESCRIPTION:     "Retrieves a specific file from a GitLab project repository.",
		TOOL_LIST_PROJECT_FILES_DESCRIPTION:   "Lists files in a directory within a GitLab project.",
		TOOL_GET_PROJECT_BRANCHES_DESCRIPTION: "Lists all branches in a GitLab project.",
		TOOL_GET_PROJECT_COMMITS_DESCRIPTION:  "Lists commits for a specific branch or ref in a GitLab project.",

		// Issues toolset
		TOOL_GET_ISSUE_DESCRIPTION:     "Retrieves details for a specific GitLab issue.",
		TOOL_LIST_ISSUES_DESCRIPTION:   "Lists GitLab issues, with optional filtering.",
		TOOL_CREATE_ISSUE_DESCRIPTION:  "Creates a new issue in a GitLab project.",
		TOOL_UPDATE_ISSUE_DESCRIPTION:  "Updates an existing GitLab issue.",
		TOOL_ISSUE_COMMENT_DESCRIPTION: "Manages comments on GitLab issues (list, create, update).",
		TOOL_GET_ISSUE_LABELS_DESCRIPTION: "Retrieves labels for a specific GitLab project.",
		TOOL_MILESTONE_DESCRIPTION:       "Manages GitLab milestones (get, create, update).",
		TOOL_LIST_MILESTONES_DESCRIPTION: "Lists milestones for a specific GitLab project.",

		// Merge Requests toolset
		TOOL_GET_MERGE_REQUEST_DESCRIPTION:      "Retrieves details for a specific GitLab merge request.",
		TOOL_LIST_MERGE_REQUESTS_DESCRIPTION:    "Lists GitLab merge requests, with optional filtering.",
		TOOL_CREATE_MERGE_REQUEST_DESCRIPTION:   "Creates a new merge request in a GitLab project.",
		TOOL_UPDATE_MERGE_REQUEST_DESCRIPTION:   "Updates an existing GitLab merge request.",
		TOOL_MERGE_REQUEST_COMMENT_DESCRIPTION:  "Manages comments on GitLab merge requests (list, create, update).",

		// Search toolset
		TOOL_SEARCH_DESCRIPTION: "Searches across GitLab resources (projects, issues, merge requests, code, milestones, etc.) with support for global, group, and project scopes.",

		// Users toolset
		TOOL_GET_CURRENT_USER_DESCRIPTION:   "Retrieves the currently authenticated user's information.",
		TOOL_GET_USER_DESCRIPTION:           "Retrieves details for a specific GitLab user.",
		TOOL_GET_USER_STATUS_DESCRIPTION:    "Retrieves the status message for a specific GitLab user.",
		TOOL_LIST_USERS_DESCRIPTION:         "Lists GitLab users with optional search and filtering.",
		TOOL_LIST_PROJECT_USERS_DESCRIPTION: "Lists users who are members of a specific GitLab project.",
		TOOL_MANAGE_USER_STATE_DESCRIPTION:  "Manages user state (block/unblock, ban/unban, activate/deactivate, approve). Admin only.",

		// Token management toolset
		TOOL_LIST_TOKENS_DESCRIPTION:       "Lists all configured GitLab tokens with their metadata.",
		TOOL_ADD_TOKEN_DESCRIPTION:         "Adds a new GitLab token configuration.",
		TOOL_UPDATE_TOKEN_DESCRIPTION:      "Updates an existing GitLab token.",
		TOOL_REMOVE_TOKEN_DESCRIPTION:      "Removes a GitLab token configuration.",
		TOOL_VALIDATE_TOKEN_DESCRIPTION:    "Validates a GitLab token by checking with the API.",
		TOOL_GET_NOTIFICATIONS_DESCRIPTION: "Retrieves notifications and warnings.",

		// Tags toolset
		TOOL_TAG_DESCRIPTION:                 "Manages GitLab repository tags (get, create, delete, getCommit).",
		TOOL_LIST_REPOSITORY_TAGS_DESCRIPTION: "Lists all tags in a GitLab repository.",

		// Pipeline Jobs toolset
		TOOL_PIPELINE_JOB_DESCRIPTION:   "Manages CI/CD pipeline jobs (list, get, trace).",
		TOOL_PIPELINE_DESCRIPTION:        "Controls GitLab CI/CD pipelines (cancel, retry).",
		TOOL_RETRY_PIPELINE_JOB_DESCRIPTION: "Retries a failed job in a pipeline.",
		TOOL_PLAY_PIPELINE_JOB_DESCRIPTION:  "Triggers a manual job in a pipeline.",
	}
}
