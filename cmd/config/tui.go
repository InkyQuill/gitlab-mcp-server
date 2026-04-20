// Package config provides TUI for managing GitLab MCP configuration.
package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Styles for TUI
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::FA")). // rose
			Background(lipgloss.Color("::25")). // background
			Padding(0, 2)

	//nolint:unused // part of the style palette; reserved for future screens
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::FA")). // rose
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("::FA")). // rose
				Background(lipgloss.Color("::25")). // background
				Padding(0, 1)

	//nolint:unused // part of the style palette; reserved for future screens
	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::DA")). // text
			Padding(0, 1)

	dimItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::55")). // subtle
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::GB")). // green
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::05")). // red
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::FA")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::DA")).
			Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("::55")).
			Italic(true)
)

// screen represents the current TUI screen
type screen int

const (
	mainMenuScreen screen = iota
	serverListScreen
	addServerScreen
	editServerScreen
	deleteConfirmScreen
)

// Messages for TUI
type (
	serverLoadedMsg []*config.ServerConfig
	serverSavedMsg  struct{}
	statusMsg       string
	backMsg         struct{}
	editServerMsg   *config.ServerConfig
	deleteServerMsg string
)

// model is the TUI model
type model struct {
	manager        *config.Manager
	screen         screen // current screen
	mainMenu       list.Model
	serverList     list.Model
	serverInput    textinput.Model
	tokenInput     textinput.Model
	hostInput      textinput.Model
	readOnlyCheck  bool
	statusMessage  string
	isError        bool
	quitting       bool
	editingServer  *config.ServerConfig // nil = adding new
	viewportHeight int
	viewportWidth  int
}

// mainMenuItem represents an item in the main menu
type mainMenuItem struct {
	title, desc string
}

func (m mainMenuItem) Title() string       { return m.title }
func (m mainMenuItem) Description() string { return m.desc }
func (m mainMenuItem) FilterValue() string { return m.title }

// serverItem represents a server in the server list
type serverItem struct {
	server *config.ServerConfig
}

func (s serverItem) Title() string {
	title := s.server.Name
	if s.server.IsDefault {
		title += " (default)"
	}
	return title
}

func (s serverItem) Description() string {
	user := s.server.Username
	if user == "" {
		user = "-"
	}
	readOnly := "no"
	if s.server.ReadOnly {
		readOnly = "yes"
	}
	return fmt.Sprintf("%s | User: %s | ReadOnly: %s",
		s.server.Host, user, readOnly)
}

func (s serverItem) FilterValue() string { return s.server.Name }

// initialModel creates the initial TUI model
func initialModel(manager *config.Manager) model {
	// Create main menu items
	items := []list.Item{
		mainMenuItem{title: "List Servers", desc: "Show all configured GitLab servers"},
		mainMenuItem{title: "Add Server", desc: "Add a new GitLab server configuration"},
		mainMenuItem{title: "Edit Server", desc: "Edit existing server configuration"},
		mainMenuItem{title: "Remove Server", desc: "Remove a server configuration"},
		mainMenuItem{title: "Set Default", desc: "Set the default server"},
		mainMenuItem{title: "Validate Tokens", desc: "Validate all server tokens"},
		mainMenuItem{title: "Quit", desc: "Exit configuration"},
	}

	// Create main menu list
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = dimItemStyle

	mainMenu := list.New(items, delegate, 0, 7)
	mainMenu.Title = "GitLab MCP Server Configuration"
	mainMenu.SetShowStatusBar(false)
	mainMenu.SetFilteringEnabled(false)
	mainMenu.Styles.Title = titleStyle
	mainMenu.Styles.PaginationStyle = dimItemStyle
	mainMenu.Styles.HelpStyle = dimItemStyle

	// Create input fields
	serverInput := textinput.New()
	serverInput.Placeholder = "Server Name"
	serverInput.CharLimit = 50
	serverInput.Width = 50

	tokenInput := textinput.New()
	tokenInput.Placeholder = "GitLab Personal Access Token"
	tokenInput.CharLimit = 100
	tokenInput.Width = 50
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'

	hostInput := textinput.New()
	hostInput.Placeholder = "GitLab Host (default: https://gitlab.com)"
	hostInput.CharLimit = 100
	hostInput.Width = 50

	// Create empty server list
	serverList := createServerList(nil)

	return model{
		manager:       manager,
		screen:        mainMenuScreen,
		mainMenu:      mainMenu,
		serverList:    serverList,
		serverInput:   serverInput,
		tokenInput:    tokenInput,
		hostInput:     hostInput,
		readOnlyCheck: false,
	}
}

func createServerList(servers []*config.ServerConfig) list.Model {
	items := make([]list.Item, len(servers))
	for i, s := range servers {
		items[i] = serverItem{server: s}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = dimItemStyle

	l := list.New(items, delegate, 0, len(items)+2)
	l.Title = "Configured Servers"
	l.SetShowStatusBar(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = dimItemStyle

	return l
}

// Init initializes the TUI
func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadServers(m.manager),
		textinput.Blink,
	)
}

// Update handles TUI updates
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen != mainMenuScreen {
				m.screen = mainMenuScreen
				m.statusMessage = ""
				m.isError = false
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.screen != mainMenuScreen {
				m.screen = mainMenuScreen
				m.statusMessage = ""
				m.isError = false
				return m, nil
			}
		}

		// Handle screen-specific updates
		switch m.screen {
		case mainMenuScreen:
			return m.updateMainMenu(msg)
		case serverListScreen:
			return m.updateServerList(msg)
		case addServerScreen, editServerScreen:
			return m.updateAddEditScreen(msg)
		case deleteConfirmScreen:
			return m.updateDeleteConfirm(msg)
		}

	case tea.WindowSizeMsg:
		m.viewportHeight = msg.Height
		m.viewportWidth = msg.Width
		return m, nil

	case serverLoadedMsg:
		m.serverList = createServerList(msg)
		return m, nil

	case serverSavedMsg:
		m.statusMessage = "Server saved successfully!"
		m.isError = false
		m.screen = serverListScreen
		m.editingServer = nil
		m.serverInput.SetValue("")
		m.tokenInput.SetValue("")
		m.hostInput.SetValue("")
		m.readOnlyCheck = false
		return m, loadServers(m.manager)

	case error:
		m.statusMessage = "Error: " + msg.Error()
		m.isError = true
		return m, nil

	case statusMsg:
		m.statusMessage = string(msg)
		m.isError = false
		return m, nil

	case backMsg:
		m.screen = mainMenuScreen
		m.statusMessage = ""
		m.isError = false
		return m, nil

	case editServerMsg:
		m.editingServer = msg
		if msg != nil {
			m.serverInput.SetValue(msg.Name)
			m.hostInput.SetValue(msg.Host)
			m.tokenInput.SetValue(msg.Token)
			m.readOnlyCheck = msg.ReadOnly
		}
		m.screen = editServerScreen
		m.serverInput.Focus()
		return m, nil

	case deleteServerMsg:
		return m, func() tea.Msg {
			if err := m.manager.RemoveServer(string(msg)); err != nil {
				return err
			}
			// Save the config
			if err := m.manager.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			return statusMsg(fmt.Sprintf("Server '%s' removed", msg))
		}
	}

	return m, nil
}

func (m model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle keys
	switch msg.String() {
	case "enter", " ":
		switch m.mainMenu.Index() {
		case 0: // List Servers
			m.screen = serverListScreen
			m.serverList.SetFilteringEnabled(true)
			return m, loadServers(m.manager)
		case 1: // Add Server
			m.editingServer = nil
			m.serverInput.Reset()
			m.tokenInput.Reset()
			m.hostInput.Reset()
			m.readOnlyCheck = false
			m.screen = addServerScreen
			m.serverInput.Focus()
			return m, textinput.Blink
		case 2: // Edit Server
			servers := m.manager.ListServers()
			if len(servers) == 0 {
				m.statusMessage = "No servers configured. Add a server first."
				m.isError = true
				return m, nil
			}
			m.serverList = createServerList(servers)
			m.screen = serverListScreen
			m.statusMessage = "Select a server to edit (press 'e')"
			m.serverList.SetFilteringEnabled(true)
			return m, loadServers(m.manager)
		case 3: // Remove Server
			servers := m.manager.ListServers()
			if len(servers) == 0 {
				m.statusMessage = "No servers configured. Add a server first."
				m.isError = true
				return m, nil
			}
			m.screen = deleteConfirmScreen
			m.statusMessage = "Select a server to remove (press 'd')"
			m.serverList.SetFilteringEnabled(true)
			return m, loadServers(m.manager)
		case 4: // Set Default
			return m, setDefaultServerCmd(m.manager)
		case 5: // Validate Tokens
			return m, validateAllCmd(m.manager)
		case 6: // Quit
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.mainMenu, cmd = m.mainMenu.Update(msg)
	return m, cmd
}

func (m model) updateServerList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		i, ok := m.serverList.SelectedItem().(serverItem)
		if ok {
			m.statusMessage = fmt.Sprintf("Server: %s | Host: %s | User: %s",
				i.server.Name, i.server.Host, i.server.Username)
			m.isError = false
		}
		return m, nil
	case "e", "E":
		i, ok := m.serverList.SelectedItem().(serverItem)
		if ok {
			// Get the full server config
			servers := m.manager.ListServers()
			for _, s := range servers {
				if s.Name == i.server.Name {
					return m, func() tea.Msg { return editServerMsg(s) }
				}
			}
		}
	case "d", "D":
		i, ok := m.serverList.SelectedItem().(serverItem)
		if ok {
			// Check if it's the default server
			if i.server.IsDefault && m.manager.ServerCount() > 1 {
				m.statusMessage = "Cannot remove default server. Set another as default first."
				m.isError = true
				return m, nil
			}
			return m, func() tea.Msg {
				return deleteServerMsg(i.server.Name)
			}
		}
	case "s", "S":
		return m, setDefaultServerCmd(m.manager)
	case "b", "B":
		m.screen = mainMenuScreen
		m.statusMessage = ""
		m.serverList.SetFilteringEnabled(false)
		return m, nil
	}

	var cmd tea.Cmd
	m.serverList, cmd = m.serverList.Update(msg)
	return m, cmd
}

func (m model) updateAddEditScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle field navigation
	switch msg.String() {
	case "tab", "shift+tab":
		// Toggle focus between inputs
		if m.serverInput.Focused() {
			m.serverInput.Blur()
			m.tokenInput.Focus()
		} else if m.tokenInput.Focused() {
			m.tokenInput.Blur()
			m.hostInput.Focus()
		} else {
			m.hostInput.Blur()
			m.serverInput.Focus()
		}
		return m, nil
	case "r", "R":
		m.readOnlyCheck = !m.readOnlyCheck
		return m, nil
	case "enter":
		// Validate inputs
		name := strings.TrimSpace(m.serverInput.Value())
		token := strings.TrimSpace(m.tokenInput.Value())
		host := strings.TrimSpace(m.hostInput.Value())

		if name == "" {
			m.statusMessage = "Server name is required"
			m.isError = true
			return m, nil
		}
		if token == "" {
			m.statusMessage = "Token is required"
			m.isError = true
			return m, nil
		}

		serverCfg := &config.ServerConfig{
			Name:     name,
			Token:    token,
			Host:     host,
			ReadOnly: m.readOnlyCheck,
		}

		return m, saveServerCmd(m.manager, serverCfg, m.editingServer != nil)
	case "esc":
		m.screen = mainMenuScreen
		m.statusMessage = ""
		m.editingServer = nil
		m.serverInput.Blur()
		m.tokenInput.Blur()
		m.hostInput.Blur()
		return m, nil
	}

	// Update focused input
	if m.serverInput.Focused() {
		m.serverInput, cmd = m.serverInput.Update(msg)
		return m, cmd
	}
	if m.tokenInput.Focused() {
		m.tokenInput, cmd = m.tokenInput.Update(msg)
		return m, cmd
	}
	m.hostInput, cmd = m.hostInput.Update(msg)
	return m, cmd
}

func (m model) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "d", "D", "y", "Y":
		i, ok := m.serverList.SelectedItem().(serverItem)
		if ok {
			// Check if it's the default server
			if i.server.IsDefault && m.manager.ServerCount() > 1 {
				m.statusMessage = "Cannot remove default server. Set another as default first."
				m.isError = true
				return m, nil
			}
			return m, func() tea.Msg {
				return deleteServerMsg(i.server.Name)
			}
		}
	case "n", "N", "esc":
		m.screen = mainMenuScreen
		m.statusMessage = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.serverList, cmd = m.serverList.Update(msg)
	return m, cmd
}

// View renders the TUI
func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var content string

	switch m.screen {
	case mainMenuScreen:
		content = m.viewMainMenu()
	case serverListScreen:
		content = m.viewServerList()
	case addServerScreen, editServerScreen:
		content = m.viewAddEditScreen()
	case deleteConfirmScreen:
		content = m.viewDeleteConfirm()
	}

	// Add status bar if there's a message
	if m.statusMessage != "" {
		style := statusStyle
		if m.isError {
			style = errorStyle
		}
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			"\n",
			style.Render(m.statusMessage),
		)
	}

	// Add help text
	help := m.getHelpText()
	content = lipgloss.JoinVertical(lipgloss.Left,
		content,
		"\n",
		hintStyle.Render(help),
	)

	// Center content if we have viewport dimensions
	if m.viewportWidth > 0 && m.viewportHeight > 0 {
		content = lipgloss.Place(
			m.viewportWidth, m.viewportHeight-3,
			lipgloss.Center, lipgloss.Center,
			content,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("::bg")),
		)
	} else {
		// Default width if no viewport
		content = lipgloss.Place(
			60, 20,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}

	return "\n" + content
}

func (m model) viewMainMenu() string {
	return m.mainMenu.View()
}

func (m model) viewServerList() string {
	if len(m.serverList.Items()) == 0 {
		return dimItemStyle.Render("No servers configured.\n\nPress 'a' to add a server or 'esc' to go back.")
	}
	return m.serverList.View()
}

func (m model) viewAddEditScreen() string {
	title := "Add Server"
	if m.screen == editServerScreen {
		title = "Edit Server"
	}

	// Determine which field has focus
	serverField := m.serverInput.View()
	tokenField := m.tokenInput.View()
	hostField := m.hostInput.View()

	if m.serverInput.Focused() {
		serverField = selectedItemStyle.Render(serverField)
	} else {
		serverField = inputStyle.Render(serverField)
	}

	if m.tokenInput.Focused() {
		tokenField = selectedItemStyle.Render(tokenField)
	} else {
		tokenField = inputStyle.Render(tokenField)
	}

	if m.hostInput.Focused() {
		hostField = selectedItemStyle.Render(hostField)
	} else {
		hostField = inputStyle.Render(hostField)
	}

	// Read-only checkbox
	readOnlyStr := "[ ]"
	if m.readOnlyCheck {
		readOnlyStr = "[x]"
	}
	if m.readOnlyCheck {
		readOnlyStr = selectedItemStyle.Render(readOnlyStr + " Read-Only Mode")
	} else {
		readOnlyStr = inputStyle.Render(readOnlyStr + " Read-Only Mode")
	}

	fields := []string{
		titleStyle.Width(60).Render(title),
		"",
		labelStyle.Render("Server Name:"),
		serverField,
		"",
		labelStyle.Render("GitLab Host:"),
		hostField,
		"",
		labelStyle.Render("Personal Access Token:"),
		tokenField,
		"",
		readOnlyStr,
		"",
	}

	return strings.Join(fields, "\n")
}

func (m model) viewDeleteConfirm() string {
	if len(m.serverList.Items()) == 0 {
		return dimItemStyle.Render("No servers configured.")
	}

	i, ok := m.serverList.SelectedItem().(serverItem)
	if !ok {
		return dimItemStyle.Render("No server selected.")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.serverList.View(),
		"",
		errorStyle.Render(fmt.Sprintf("Press 'd' to remove '%s' or 'esc' to cancel", i.server.Name)),
	)
}

func (m model) getHelpText() string {
	switch m.screen {
	case mainMenuScreen:
		return "enter/↑↓: select | ctrl+c/q: quit"
	case serverListScreen:
		return "enter: view | e: edit | d: remove | s: set default | b: back | esc: main menu"
	case addServerScreen, editServerScreen:
		return "tab: next field | r: toggle read-only | enter: save | esc: cancel"
	case deleteConfirmScreen:
		return "d/y: confirm removal | n/esc: cancel"
	default:
		return "esc: back | ctrl+c: quit"
	}
}

// Commands
func loadServers(manager *config.Manager) tea.Cmd {
	return func() tea.Msg {
		servers := manager.ListServers()
		return serverLoadedMsg(servers)
	}
}

func saveServerCmd(manager *config.Manager, server *config.ServerConfig, isUpdate bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Normalize host
		host := server.Host
		if host == "" {
			host = "https://gitlab.com"
		}
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "https://" + host
		}
		server.Host = host

		// Validate token by calling GitLab API
		client, err := createTuiGitLabClient(server.Host, server.Token)
		if err != nil {
			return fmt.Errorf("failed to create GitLab client: %w", err)
		}

		user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
		if err != nil {
			if resp != nil && resp.StatusCode == 401 {
				return fmt.Errorf("token validation failed: invalid or expired token (401)")
			}
			return fmt.Errorf("token validation failed: %w", err)
		}

		// Update server with user info
		server.UserID = user.ID
		server.Username = user.Username

		// Add or update server
		if isUpdate {
			// For update, remove the old server first
			if err := manager.RemoveServer(server.Name); err != nil {
				return fmt.Errorf("failed to update server: %w", err)
			}
		}

		if err := manager.AddServer(server); err != nil {
			return fmt.Errorf("failed to add server: %w", err)
		}

		// Save config
		if err := manager.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		return serverSavedMsg{}
	}
}

func setDefaultServerCmd(manager *config.Manager) tea.Cmd {
	return func() tea.Msg {
		// Get servers
		servers := manager.ListServers()
		if len(servers) == 0 {
			return fmt.Errorf("no servers configured")
		}

		// Find the default or use the first one
		var defaultName string
		for _, s := range servers {
			if s.IsDefault {
				defaultName = s.Name
				break
			}
		}

		if defaultName == "" && len(servers) > 0 {
			defaultName = servers[0].Name
		}

		// If there's only one server, it's already the default
		if len(servers) == 1 {
			return statusMsg(fmt.Sprintf("'%s' is the only server (already default)", defaultName))
		}

		// For simplicity in this TUI, we'll just cycle through servers
		// Find the current default and select the next one
		for i, s := range servers {
			if s.Name == defaultName {
				nextIdx := (i + 1) % len(servers)
				nextServer := servers[nextIdx]
				if err := manager.SetDefaultServer(nextServer.Name); err != nil {
					return fmt.Errorf("failed to set default: %w", err)
				}
				if err := manager.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				return statusMsg(fmt.Sprintf("Default server set to '%s'", nextServer.Name))
			}
		}

		return statusMsg(fmt.Sprintf("'%s' is the default server", defaultName))
	}
}

func validateAllCmd(manager *config.Manager) tea.Cmd {
	return func() tea.Msg {
		results := manager.ValidateAllServers(context.Background())

		var valid, invalid int
		var failedServers []string
		for name, err := range results {
			if err == nil {
				valid++
			} else {
				invalid++
				failedServers = append(failedServers, fmt.Sprintf("%s: %v", name, err))
			}
		}

		msg := fmt.Sprintf("Validation complete: %d valid, %d invalid", valid, invalid)
		if invalid > 0 && len(failedServers) > 0 {
			msg += fmt.Sprintf("\nFailed: %s", strings.Join(failedServers[:minInt(2, len(failedServers))], "; "))
			if len(failedServers) > 2 {
				msg += "..."
			}
		}

		return statusMsg(msg)
	}
}

// createTuiGitLabClient creates a GitLab client (renamed to avoid conflict with add.go)
func createTuiGitLabClient(host, token string) (*gl.Client, error) {
	opts := []gl.ClientOptionFunc{}
	if host != "" && host != "https://gitlab.com" {
		opts = append(opts, gl.WithBaseURL(host))
	}
	return gl.NewClient(token, opts...)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
