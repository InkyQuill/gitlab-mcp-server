// Package install provides the MCP server installation command for IDEs.
package install

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompter handles interactive prompts for the user.
type Prompter struct {
	reader *bufio.Reader
}

// NewPrompter creates a new Prompter.
func NewPrompter() *Prompter {
	return &Prompter{
		reader: bufio.NewReader(os.Stdin),
	}
}

// PromptForIDE prompts the user to select an IDE from the available options.
func PromptForIDE(available []IDE) (IDE, error) {
	if len(available) == 0 {
		return IDE{}, fmt.Errorf("no IDEs available")
	}
	if len(available) == 1 {
		return available[0], nil
	}

	p := NewPrompter()

	fmt.Println("Available IDEs:")
	for i, ide := range available {
		fmt.Printf("  %d. %s\n", i+1, formatIDEName(ide.Name))
	}

	for {
		fmt.Print("\nSelect IDE (number): ")
		input, err := p.reader.ReadString('\n')
		if err != nil {
			return IDE{}, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil {
			fmt.Println("Please enter a valid number")
			continue
		}

		if choice < 1 || choice > len(available) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(available))
			continue
		}

		return available[choice-1], nil
	}
}

// PromptForConfig prompts the user if they want to create a config file.
func PromptForConfig() (bool, error) {
	p := NewPrompter()

	fmt.Println("\nNext step: Create configuration")
	fmt.Println("Would you like to create a GitLab MCP configuration file now?")
	fmt.Println("This will allow you to store your GitLab tokens securely.")

	for {
		fmt.Print("Create config? [Y/n]: ")
		input, err := p.reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" || input == "y" || input == "yes" {
			return true, nil
		}
		if input == "n" || input == "no" {
			return false, nil
		}

		fmt.Println("Please enter 'y' or 'n'")
	}
}

// PromptForProjectSetup prompts the user if they want to set up a project.
func PromptForProjectSetup() (bool, error) {
	p := NewPrompter()

	fmt.Println("\nNext step: Project configuration")
	fmt.Println("Would you like to configure a GitLab project for this directory?")
	fmt.Println("This will create a .gmcprc file for automatic project detection.")

	for {
		fmt.Print("Configure project? [Y/n]: ")
		input, err := p.reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" || input == "y" || input == "yes" {
			return true, nil
		}
		if input == "n" || input == "no" {
			return false, nil
		}

		fmt.Println("Please enter 'y' or 'n'")
	}
}

// PromptForToken prompts the user to enter a GitLab token.
func PromptForToken() (string, error) {
	p := NewPrompter()

	fmt.Print("Enter GitLab Personal Access Token: ")
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// PromptForHost prompts the user to enter a GitLab host URL.
func PromptForHost(defaultHost string) (string, error) {
	p := NewPrompter()

	fmt.Printf("Enter GitLab host [%s]: ", defaultHost)
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultHost, nil
	}
	return input, nil
}

// PromptForProjectID prompts the user to enter a project ID.
func PromptForProjectID() (string, error) {
	p := NewPrompter()

	fmt.Print("Enter GitLab project ID (e.g., 'owner/repo' or numeric ID): ")
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// PromptForYesNo prompts the user for a yes/no response.
func PromptForYesNo(prompt string, defaultValue bool) (bool, error) {
	p := NewPrompter()

	defaultStr := "Y/n"
	if !defaultValue {
		defaultStr = "y/N"
	}

	for {
		fmt.Printf("%s [%s]: ", prompt, defaultStr)
		input, err := p.reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" {
			return defaultValue, nil
		}
		if input == "y" || input == "yes" {
			return true, nil
		}
		if input == "n" || input == "no" {
			return false, nil
		}

		fmt.Println("Please enter 'y' or 'n'")
	}
}

// PromptForChoice prompts the user to select from a list of choices.
func PromptForChoice(prompt string, choices []string, defaultIndex int) (int, error) {
	p := NewPrompter()

	if len(choices) == 0 {
		return -1, fmt.Errorf("no choices provided")
	}
	if len(choices) == 1 {
		return 0, nil
	}

	fmt.Println(prompt)
	for i, choice := range choices {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Printf("  %s %d. %s\n", marker, i+1, choice)
	}

	for {
		defaultStr := ""
		if defaultIndex >= 0 && defaultIndex < len(choices) {
			defaultStr = fmt.Sprintf(" [%d]", defaultIndex+1)
		}
		fmt.Printf("Select choice%s: ", defaultStr)

		input, err := p.reader.ReadString('\n')
		if err != nil {
			return -1, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" && defaultIndex >= 0 {
			return defaultIndex, nil
		}

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil {
			fmt.Println("Please enter a valid number")
			continue
		}

		if choice < 1 || choice > len(choices) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(choices))
			continue
		}

		return choice - 1, nil
	}
}

// ConfirmAction prompts the user to confirm an action.
func ConfirmAction(action string) (bool, error) {
	p := NewPrompter()

	fmt.Printf("%s [y/N]: ", action)
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

// formatIDEName formats the IDE name for display.
func formatIDEName(name string) string {
	switch name {
	case IDEClaudeDesktop:
		return "Claude Desktop"
	case IDEVSCode:
		return "Visual Studio Code"
	case IDECursor:
		return "Cursor"
	default:
		return name
	}
}

// PrintHeader prints a formatted header.
func PrintHeader(title string) {
	width := len(title) + 4
	fmt.Println()
	for i := 0; i < width; i++ {
		fmt.Print("─")
	}
	fmt.Println()
	fmt.Printf("  %s\n", title)
	for i := 0; i < width; i++ {
		fmt.Print("─")
	}
	fmt.Println()
}

// PrintSuccess prints a success message.
func PrintSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

// PrintError prints an error message.
func PrintError(message string) {
	fmt.Printf("✗ %s\n", message)
}

// PrintInfo prints an info message.
func PrintInfo(message string) {
	fmt.Printf("ℹ %s\n", message)
}

// PrintWarning prints a warning message.
func PrintWarning(message string) {
	fmt.Printf("⚠ %s\n", message)
}

// InteractiveInstallOptions holds options for interactive installation.
type InteractiveInstallOptions struct {
	Editor      string
	BinaryPath  string
	ConfigPath  string
	CreateConfig bool
	SetupProject bool
}

// RunInteractiveInstall runs the installation interactively.
func RunInteractiveInstall(opts *InteractiveInstallOptions) error {
	PrintHeader("GitLab MCP Server Installation")

	// Step 1: Detect IDEs
	detector, err := NewIDEDetector()
	if err != nil {
		return fmt.Errorf("failed to create IDE detector: %w", err)
	}

	allIDEs := detector.Detect()
	var availableIDEs []IDE
	for _, ide := range allIDEs {
		if ide.Enabled {
			availableIDEs = append(availableIDEs, ide)
		}
	}

	if len(availableIDEs) == 0 {
		PrintError("No supported IDEs detected")
		fmt.Println("\nSupported IDEs:")
		fmt.Println("  - Claude Desktop")
		fmt.Println("  - Visual Studio Code")
		fmt.Println("  - Cursor")
		return fmt.Errorf("no IDEs available for installation")
	}

	PrintInfo(fmt.Sprintf("Detected %d IDE(s)", len(availableIDEs)))

	// Step 2: Select IDE (if not specified)
	var selectedIDE IDE
	if opts.Editor != "" {
		for _, ide := range availableIDEs {
			if ide.Name == opts.Editor {
				selectedIDE = ide
				break
			}
		}
		if selectedIDE.Name == "" {
			return fmt.Errorf("specified IDE '%s' not detected", opts.Editor)
		}
	} else {
		selectedIDE, err = PromptForIDE(availableIDEs)
		if err != nil {
			return fmt.Errorf("failed to select IDE: %w", err)
		}
	}

	PrintSuccess(fmt.Sprintf("Selected: %s", formatIDEName(selectedIDE.Name)))

	// Step 3: Get binary path
	binaryPath, err := getSelfExecutablePath(opts.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to determine binary path: %w", err)
	}
	PrintInfo(fmt.Sprintf("Binary path: %s", binaryPath))

	// Step 4: Perform installation
	PrintInfo(fmt.Sprintf("Installing to: %s", selectedIDE.ConfigPath))

	installConfig := createInstallConfig(binaryPath, opts.ConfigPath)
	if err := installForIDE(selectedIDE, installConfig, false); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	PrintSuccess("Installation complete!")

	// Step 5: Prompt for additional setup
	if opts.CreateConfig {
		createConfig, err := PromptForConfig()
		if err != nil {
			PrintWarning(fmt.Sprintf("Failed to prompt for config: %v", err))
		} else if createConfig {
			fmt.Println("\nTo create a config, run:")
			fmt.Println("  gitlab-mcp-server config init")
		}
	}

	if opts.SetupProject {
		setupProject, err := PromptForProjectSetup()
		if err != nil {
			PrintWarning(fmt.Sprintf("Failed to prompt for project: %v", err))
		} else if setupProject {
			fmt.Println("\nTo set up a project, run:")
			fmt.Println("  gitlab-mcp-server project init")
		}
	}

	return nil
}
