package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Set up Claude to use Boba",
	RunE:  runInstall,
}

var (
	flagDesktopOnly bool
	flagCodeOnly    bool
)

func init() {
	installCmd.Flags().BoolVar(&flagDesktopOnly, "desktop-only", false, "Only install for Claude Desktop")
	installCmd.Flags().BoolVar(&flagCodeOnly, "code-only", false, "Only install for Claude Code")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Use the PATH-resolved location (e.g. /Users/x/.nvm/.../bin/boba)
	// rather than os.Executable() which resolves deep into node_modules
	// and breaks when npm reorganizes on updates.
	binaryPath, err := exec.LookPath("boba")
	if err != nil {
		// Fallback to os.Executable if not in PATH
		binaryPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine binary path: %w", err)
		}
	}
	binaryPath, _ = filepath.Abs(binaryPath)

	// On Windows, npm installs a .cmd wrapper. Claude Desktop can't
	// execute .cmd files directly — wrap with cmd.exe /c.
	var mcpCommand string
	var mcpArgs []string
	if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(binaryPath), ".cmd") {
		mcpCommand = "cmd.exe"
		mcpArgs = []string{"/c", binaryPath, "mcp"}
	} else {
		mcpCommand = binaryPath
		mcpArgs = []string{"mcp"}
	}

	var desktopErr, codeErr error
	desktopSkipped := flagCodeOnly
	codeSkipped := flagDesktopOnly

	if !desktopSkipped {
		desktopErr = installDesktop(mcpCommand, mcpArgs)
	}
	if !codeSkipped {
		codeErr = installCode(mcpCommand, mcpArgs)
	}

	lines := buildInstallLines(mcpCommand, mcpArgs, desktopErr, codeErr, desktopSkipped, codeSkipped)
	runScanReveal(lines)

	return nil
}

func buildInstallLines(mcpCommand string, mcpArgs []string, desktopErr, codeErr error, desktopSkipped, codeSkipped bool) []string {
	var lines []string

	for _, l := range strings.Split(ui.RenderLogo(), "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	check := lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("✓")
	skip := lipgloss.NewStyle().Foreground(ui.ColorDim).Render("○")
	cross := lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true).Render("✗")
	dim := ui.DimStyle

	if desktopSkipped {
		lines = append(lines, "  "+skip+" "+dim.Render("Claude Desktop")+"  "+dim.Render("skipped"))
	} else if desktopErr != nil {
		lines = append(lines, "  "+cross+" "+dim.Render("Claude Desktop")+"  "+ui.ErrorStyle.Render(desktopErr.Error()))
	} else {
		lines = append(lines, "  "+check+" "+dim.Render("Claude Desktop")+"  "+ui.SuccessStyle.Render("installed"))
	}

	if codeSkipped {
		lines = append(lines, "  "+skip+" "+dim.Render("Claude Code")+"     "+dim.Render("skipped"))
	} else if codeErr != nil {
		lines = append(lines, "  "+cross+" "+dim.Render("Claude Code")+"     "+ui.ErrorStyle.Render(codeErr.Error()))
	} else {
		lines = append(lines, "  "+check+" "+dim.Render("Claude Code")+"     "+ui.SuccessStyle.Render("installed"))
	}

	lines = append(lines, "")

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ui.ColorBoba).
		Bold(true).
		Padding(0, 2)

	label := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(14)
	val := lipgloss.NewStyle().Foreground(ui.ColorPearl)

	rows := []string{
		fmt.Sprintf("  %s %s", label.Render("Binary"), val.Render(mcpCommand)),
		fmt.Sprintf("  %s %s", label.Render("Args"), val.Render(strings.Join(mcpArgs, " "))),
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorDim).
		Padding(1, 2).
		Render(
			headerStyle.Render(" MCP CONFIG ") + "\n\n" +
				strings.Join(rows, "\n"))

	for _, l := range strings.Split(card, "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	num := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ui.ColorBoba).
		Bold(true).
		Padding(0, 1)

	lines = append(lines, "  "+dim.Render("Next steps:"))
	lines = append(lines, "  "+num.Render("1")+" "+ui.BrightStyle.Render("boba start")+"   "+dim.Render("Start the proxy server"))
	lines = append(lines, "  "+num.Render("2")+" "+ui.BrightStyle.Render("boba launch")+"  "+dim.Render("Start proxy and open Claude"))
	lines = append(lines, "")

	return lines
}

func installDesktop(command string, args []string) error {
	var configPath string
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		configPath = filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	default:
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".config", "claude", "claude_desktop_config.json")
	}

	return writeMCPConfig(configPath, command, args)
}

func installCode(command string, args []string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claude.json")

	return writeCodeConfig(configPath, command, args)
}

func writeMCPConfig(configPath, command string, args []string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var existing map[string]any
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]any)
	}

	mcpServers, ok := existing["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	mcpServers["boba"] = map[string]any{
		"command": command,
		"args":    args,
	}

	existing["mcpServers"] = mcpServers

	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, output, 0644)
}

func writeCodeConfig(configPath, command string, args []string) error {
	var existing map[string]any
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]any)
	}

	mcpServers, ok := existing["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	mcpServers["boba"] = map[string]any{
		"command": command,
		"args":    args,
	}

	existing["mcpServers"] = mcpServers

	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, output, 0644)
}
