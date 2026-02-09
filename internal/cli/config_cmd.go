package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Change your settings",
	RunE:  runConfig,
}

var (
	flagMCPURL  string
	flagAuthURL string
	flagCfgPort string
	flagReset   bool
	flagForce   bool
)

func init() {
	configCmd.Flags().StringVar(&flagMCPURL, "mcp-url", "", "Set MCP server URL")
	configCmd.Flags().StringVar(&flagAuthURL, "auth-url", "", "Set auth server URL")
	configCmd.Flags().StringVar(&flagCfgPort, "port", "", "Set default proxy port")
	configCmd.Flags().BoolVar(&flagReset, "reset", false, "Reset all config to defaults")
	configCmd.Flags().BoolVar(&flagForce, "force", false, "Skip URL validation")
}

func runConfig(cmd *cobra.Command, args []string) error {
	if flagReset {
		if err := config.Reset(); err != nil {
			return fmt.Errorf("failed to reset config: %w", err)
		}
	}

	changed := false

	if flagMCPURL != "" {
		if err := config.SetMCPURL(flagMCPURL, flagForce); err != nil {
			return err
		}
		changed = true
	}

	if flagAuthURL != "" {
		if err := config.SetAuthURL(flagAuthURL, flagForce); err != nil {
			return err
		}
		changed = true
	}

	if flagCfgPort != "" {
		port, err := strconv.Atoi(flagCfgPort)
		if err != nil {
			return fmt.Errorf("invalid port: %s", flagCfgPort)
		}
		if err := config.SetProxyPort(port); err != nil {
			return fmt.Errorf("failed to set port: %w", err)
		}
		changed = true
	}

	lines := buildConfigLines(flagReset, changed)
	runScanReveal(lines)

	return nil
}

func buildConfigLines(wasReset, wasChanged bool) []string {
	var lines []string

	for _, l := range strings.Split(ui.RenderLogo(), "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	if wasReset {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorGreen).
			Bold(true).
			Padding(0, 2).
			Render(" RESET COMPLETE ")
		lines = append(lines, "  "+badge+"  "+ui.DimStyle.Render("All settings restored to defaults"))
		lines = append(lines, "")
	} else if wasChanged {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorGreen).
			Bold(true).
			Padding(0, 2).
			Render(" UPDATED ")
		lines = append(lines, "  "+badge+"  "+ui.SuccessStyle.Render("Configuration saved ✓"))
		lines = append(lines, "")
	}

	cfgHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ui.ColorDim).
		Bold(true).
		Padding(0, 2)

	label := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(14)
	val := lipgloss.NewStyle().Foreground(ui.ColorPearl)

	configRows := []string{
		fmt.Sprintf("  %s %s", label.Render("MCP URL"), val.Render(config.GetMCPURL())),
		fmt.Sprintf("  %s %s", label.Render("Auth URL"), val.Render(config.GetAuthURL())),
		fmt.Sprintf("  %s %s", label.Render("Proxy Port"), val.Render(fmt.Sprintf("%d", config.GetProxyPort()))),
		fmt.Sprintf("  %s %s", label.Render("Log Level"), val.Render(config.GetLogLevel())),
		fmt.Sprintf("  %s %s", label.Render("Config"), val.Render(config.ConfigPath())),
	}

	configCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorDim).
		Padding(1, 2).
		Render(
			cfgHeader.Render(" CONFIGURATION ") + "\n\n" +
				strings.Join(configRows, "\n"))

	for _, l := range strings.Split(configCard, "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	return lines
}
