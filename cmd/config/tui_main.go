// Package config provides TUI main entry point for managing GitLab MCP configuration.
package config

import (
	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/charmbracelet/bubbletea"
)

// RunTUI launches the interactive TUI for managing GitLab MCP configuration.
// It returns an error if the TUI fails to start or run.
func RunTUI(cfgManager *config.Manager) error {
	p := tea.NewProgram(
		initialModel(cfgManager),
		tea.WithAltScreen(),       // Use alternate screen
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if user quit normally
	if m, ok := finalModel.(model); ok && m.quitting {
		return nil
	}

	return nil
}
