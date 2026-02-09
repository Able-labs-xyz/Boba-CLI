package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/auth"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Test your connection",
	RunE:  runAuth,
}

func runAuth(cmd *cobra.Command, args []string) error {
	if !config.HasCredentials() {
		return fmt.Errorf("no credentials configured. Run 'boba login' first")
	}

	ui.PrintLogo()
	fmt.Println()

	var tokens *config.AuthTokens
	err := ui.RunWithSpinner("Authenticating with Boba network...", func() error {
		var authErr error
		tokens, authErr = auth.Authenticate()
		return authErr
	})
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorBox("Authentication failed: " + err.Error()))
		return err
	}

	fmt.Println()

	lines := buildAuthResultLines(tokens)
	runScanReveal(lines)

	return nil
}

func buildAuthResultLines(tokens *config.AuthTokens) []string {
	var lines []string

	agentDisplay := tokens.AgentName
	if agentDisplay == "" {
		agentDisplay = tokens.AgentID
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ui.ColorGreen).
		Bold(true).
		Padding(0, 2)

	dimLabel := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(14)
	brightVal := lipgloss.NewStyle().Foreground(ui.ColorBright)
	greenDot := lipgloss.NewStyle().Foreground(ui.ColorGreen).Render("●")

	rows := []string{
		fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Agent"), ui.GoldStyle.Render(agentDisplay)),
		fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Agent ID"), brightVal.Render(truncateAddr(tokens.AgentID))),
		fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("EVM"), brightVal.Render(truncateAddr(tokens.EVMAddress))),
	}
	if tokens.SolanaAddress != "" {
		rows = append(rows,
			fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Solana"), brightVal.Render(truncateAddr(tokens.SolanaAddress))))
	}
	rows = append(rows,
		fmt.Sprintf("  %s %s %s", greenDot, dimLabel.Render("Expiry"), ui.DimStyle.Render(tokens.AccessTokenExpiresAt)))

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorGreen).
		Padding(1, 2).
		Render(
			headerStyle.Render(" AUTH SUCCESS ") + "\n\n" +
				strings.Join(rows, "\n"))

	for _, l := range strings.Split(card, "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	return lines
}
