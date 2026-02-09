package cli

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out",
	RunE:  runLogout,
}

type logoutTickMsg struct{}
type logoutModel struct {
	steps   []string
	current int
	done    bool
	err     error
	frame   int
}

var logoutSteps = []string{
	"Clearing credentials from keychain",
	"Revoking auth tokens",
	"Destroying session token",
	"Purging configuration",
}

func logoutTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(_ time.Time) tea.Msg { return logoutTickMsg{} })
}

func (m logoutModel) Init() tea.Cmd {
	return logoutTick()
}

func (m logoutModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.done = true
		m.current = len(m.steps)
		return m, tea.Quit
	case logoutTickMsg:
		m.frame++

		// Actually clear on the first tick
		if m.current == 0 && m.frame == 1 {
			if err := config.ClearCredentials(); err != nil {
				m.err = err
				return m, tea.Quit
			}
		}

		m.current++
		if m.current >= len(m.steps) {
			m.done = true
			return m, tea.Quit
		}
		return m, logoutTick()
	}
	return m, nil
}

func (m logoutModel) View() string {
	var b strings.Builder

	headerText := "DISCONNECTING..."
	headerProgress := float64(m.frame) / 6.0
	if headerProgress > 1.0 {
		headerProgress = 1.0
	}
	glitchedHeader := ui.GlitchText(headerText, headerProgress)
	headerStyle := lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true)
	b.WriteString("\n  " + headerStyle.Render(glitchedHeader))
	b.WriteString("\n\n")

	checkStyle := lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true)
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	for i, step := range m.steps {
		if i < m.current {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				checkStyle.Render("✓"),
				lipgloss.NewStyle().Foreground(ui.ColorDim).Render(step)))
		} else if i == m.current {
			xChar := "✗"
			if m.frame%2 == 0 {
				xChar = "×"
			}
			glitched := ui.GlitchText(step, 0.7)
			b.WriteString(fmt.Sprintf("  %s %s\n",
				activeStyle.Render(xChar),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(glitched)))
		} else {
			scrambled := ui.GlitchText(step, 0.0)
			b.WriteString(fmt.Sprintf("    %s\n",
				pendingStyle.Render(scrambled)))
		}
	}

	// Always render the badge area (consistent height)
	b.WriteString("\n")
	if m.done {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorRed).
			Bold(true).
			Padding(0, 2).
			Render(" DISCONNECTED ")
		b.WriteString("  " + badge)
	}
	b.WriteString("\n")

	return b.String()
}

func runLogout(cmd *cobra.Command, args []string) error {
	ui.PrintLogo()
	fmt.Println()

	model := logoutModel{steps: logoutSteps}
	p := tea.NewProgram(model, tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("logout animation failed: %w", err)
	}

	m := finalModel.(logoutModel)
	if m.err != nil {
		return fmt.Errorf("failed to clear credentials: %w", m.err)
	}

	fmt.Println()
	dim := ui.DimStyle
	fmt.Println("  " + dim.Render("Run ") + ui.BrightStyle.Render("boba login") + dim.Render(" to reconnect."))
	fmt.Println()

	return nil
}
