package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "See if everything's working",
	RunE:  runStatus,
}

func buildStatusLines() []string {
	var lines []string

	for _, l := range strings.Split(ui.RenderLogo(), "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ui.ColorBoba).
		Bold(true).
		Padding(0, 2)

	dimLabel := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(14)
	brightVal := lipgloss.NewStyle().Foreground(ui.ColorBright)
	greenDot := lipgloss.NewStyle().Foreground(ui.ColorGreen).Render("●")
	redDot := lipgloss.NewStyle().Foreground(ui.ColorRed).Render("●")

	var statusRows []string
	statusRows = append(statusRows, headerStyle.Render(" CONNECTION STATUS "))
	statusRows = append(statusRows, "")

	if config.HasCredentials() {
		statusRows = append(statusRows,
			fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Credentials"), ui.SuccessStyle.Render("configured ✓")))

		c := config.Load()
		if c.Credentials != nil {
			statusRows = append(statusRows,
				fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Agent ID"), brightVal.Render(truncateAddr(c.Credentials.AgentID))))
		}

		if config.IsTokenExpired() {
			statusRows = append(statusRows,
				fmt.Sprintf("  %s %s %s", redDot, dimLabel.Render("Token"), ui.ErrorStyle.Render("expired ✗")))
		} else {
			statusRows = append(statusRows,
				fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Token"), ui.SuccessStyle.Render("valid ✓")))
		}

		tokens, err := config.GetTokens()
		if err == nil {
			if tokens.AgentName != "" {
				statusRows = append(statusRows,
					fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Agent"), ui.GoldStyle.Render(tokens.AgentName)))
			}
			if tokens.EVMAddress != "" {
				statusRows = append(statusRows,
					fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("EVM"), brightVal.Render(truncateAddr(tokens.EVMAddress))))
			}
			if tokens.SolanaAddress != "" {
				statusRows = append(statusRows,
					fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Solana"), brightVal.Render(truncateAddr(tokens.SolanaAddress))))
			}
		}
	} else {
		statusRows = append(statusRows,
			fmt.Sprintf("  %s %s %s", redDot, dimLabel.Render("Credentials"), ui.ErrorStyle.Render("not initialized")))
		statusRows = append(statusRows, "")
		statusRows = append(statusRows,
			"  "+ui.DimStyle.Render("Run ")+ui.BrightStyle.Render("boba login")+ui.DimStyle.Render(" to get started"))
	}

	statusContent := strings.Join(statusRows, "\n")
	statusCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBoba).
		Padding(1, 2).
		Render(statusContent)

	for _, l := range strings.Split(statusCard, "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	cfgHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ui.ColorDim).
		Bold(true).
		Padding(0, 2)

	cfgLabel := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(14)
	cfgVal := lipgloss.NewStyle().Foreground(ui.ColorPearl)

	var cfgRows []string
	cfgRows = append(cfgRows, cfgHeader.Render(" CONFIGURATION "))
	cfgRows = append(cfgRows, "")
	cfgRows = append(cfgRows, fmt.Sprintf("  %s %s", cfgLabel.Render("MCP URL"), cfgVal.Render(config.GetMCPURL())))
	cfgRows = append(cfgRows, fmt.Sprintf("  %s %s", cfgLabel.Render("Auth URL"), cfgVal.Render(config.GetAuthURL())))
	cfgRows = append(cfgRows, fmt.Sprintf("  %s %s", cfgLabel.Render("Proxy Port"), cfgVal.Render(fmt.Sprintf("%d", config.GetProxyPort()))))
	cfgRows = append(cfgRows, fmt.Sprintf("  %s %s", cfgLabel.Render("Log Level"), cfgVal.Render(config.GetLogLevel())))
	cfgRows = append(cfgRows, fmt.Sprintf("  %s %s", cfgLabel.Render("Config"), cfgVal.Render(config.ConfigPath())))

	cfgContent := strings.Join(cfgRows, "\n")
	cfgCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorDim).
		Padding(1, 2).
		Render(cfgContent)

	for _, l := range strings.Split(cfgCard, "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	return lines
}

func runStatus(cmd *cobra.Command, args []string) error {
	lines := buildStatusLines()
	runScanReveal(lines)
	return nil
}
