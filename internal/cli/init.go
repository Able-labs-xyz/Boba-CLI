package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/auth"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var initCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"init"},
	Short:   "Log in with your Boba Agent credentials",
	RunE:    runInit,
}

var (
	flagAgentID string
	flagSecret  string
	flagName    string
)

func init() {
	initCmd.Flags().StringVarP(&flagAgentID, "agent-id", "i", "", "Agent ID")
	initCmd.Flags().StringVarP(&flagSecret, "secret", "s", "", "Agent secret")
	initCmd.Flags().StringVarP(&flagName, "name", "n", "", "Agent name (optional)")
}

// bobaTheme delegates to the shared ui.BobaTheme.
func bobaTheme() *huh.Theme { return ui.BobaTheme() }

type onboardingStep struct {
	label string
	fn    func() error
}

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepDone
	stepFailed
)

type stepDoneMsg struct{ idx int }
type stepFailMsg struct {
	idx int
	err error
}
type allDoneMsg struct{}

type onboardingModel struct {
	steps      []onboardingStep
	statuses   []stepStatus
	errors     []error
	current    int
	spinner    spinner.Model
	done       bool
	failed     bool
	width      int
	glitchTick int // increments on each spinner tick for glitch animation
	headerTick int // typewriter counter for header text
}

func newOnboardingModel(steps []onboardingStep) onboardingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorBoba)

	statuses := make([]stepStatus, len(steps))
	errors := make([]error, len(steps))

	return onboardingModel{
		steps:    steps,
		statuses: statuses,
		errors:   errors,
		current:  0,
		spinner:  s,
		width:    60,
	}
}

func (m onboardingModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runStep(0),
	)
}

func (m onboardingModel) runStep(idx int) tea.Cmd {
	if idx >= len(m.steps) {
		return func() tea.Msg { return allDoneMsg{} }
	}
	fn := m.steps[idx].fn
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)
		if err := fn(); err != nil {
			return stepFailMsg{idx: idx, err: err}
		}
		return stepDoneMsg{idx: idx}
	}
}

func (m onboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.failed = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case stepDoneMsg:
		m.statuses[msg.idx] = stepDone
		m.glitchTick = 0 // reset glitch for next step
		next := msg.idx + 1
		if next < len(m.steps) {
			m.current = next
			m.statuses[next] = stepRunning
			return m, m.runStep(next)
		}
		// All steps done
		return m, func() tea.Msg { return allDoneMsg{} }

	case stepFailMsg:
		m.statuses[msg.idx] = stepFailed
		m.errors[msg.idx] = msg.err
		m.failed = true
		return m, tea.Quit

	case allDoneMsg:
		m.done = true
		return m, tea.Quit

	default:
		m.glitchTick++
		m.headerTick++
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m onboardingModel) View() string {
	var b strings.Builder

	// Header with typewriter/glitch decrypt effect
	headerText := "ESTABLISHING SECURE CONNECTION..."
	headerProgress := float64(m.headerTick) / 25.0
	if headerProgress > 1.0 {
		headerProgress = 1.0
	}
	glitchedHeader := ui.GlitchText(headerText, headerProgress)
	headerStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)
	b.WriteString("  " + headerStyle.Render(glitchedHeader))
	b.WriteString("\n\n")

	checkmark := lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("  ✓ ")
	failmark := lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true).Render("  ✗ ")

	labelDone := lipgloss.NewStyle().Foreground(ui.ColorGreen)
	labelActive := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	labelPending := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	labelFailed := lipgloss.NewStyle().Foreground(ui.ColorRed)

	for i, step := range m.steps {
		switch m.statuses[i] {
		case stepDone:
			b.WriteString(checkmark)
			b.WriteString(labelDone.Render(step.label))
		case stepRunning:
			// Active step: glitch text that gradually resolves
			glitchProgress := float64(m.glitchTick) / 15.0
			if glitchProgress > 1.0 {
				glitchProgress = 1.0
			}
			glitchedLabel := ui.GlitchText(step.label, glitchProgress)
			b.WriteString("  ")
			b.WriteString(m.spinner.View())
			b.WriteString(" ")
			b.WriteString(labelActive.Render(glitchedLabel))
		case stepFailed:
			b.WriteString(failmark)
			b.WriteString(labelFailed.Render(step.label + " — FAILED"))
		default:
			// Pending: fully scrambled
			scrambled := ui.GlitchText(step.label, 0.0)
			b.WriteString("    ")
			b.WriteString(labelPending.Render(scrambled))
		}
		b.WriteString("\n")
	}

	// Pulsing border line
	borderLen := 40
	if m.width > 6 {
		borderLen = m.width - 6
	}
	if borderLen > 50 {
		borderLen = 50
	}
	b.WriteString("\n")
	var borderColor lipgloss.Color
	if m.glitchTick%4 < 2 {
		borderColor = ui.ColorDim
	} else {
		borderColor = ui.ColorBoba
	}
	b.WriteString("  " + lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("━", borderLen)))
	b.WriteString("\n")

	return b.String()
}

func renderSuccessCard(tokens *config.AuthTokens) string {
	var b strings.Builder

	agentDisplay := tokens.AgentName
	if agentDisplay == "" {
		agentDisplay = tokens.AgentID
	}

	tagline := lipgloss.NewStyle().
		Foreground(ui.ColorGold).
		Bold(true).
		Render("You're in. Let's trade.")

	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(10)
	valStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	dimValStyle := lipgloss.NewStyle().Foreground(ui.ColorBright)

	lines := []string{
		keyStyle.Render("Agent") + valStyle.Render(agentDisplay),
		keyStyle.Render("ID") + dimValStyle.Render(truncateAddr(tokens.AgentID)),
		keyStyle.Render("EVM") + dimValStyle.Render(truncateAddr(tokens.EVMAddress)),
	}
	if tokens.SolanaAddress != "" {
		lines = append(lines, keyStyle.Render("Solana")+dimValStyle.Render(truncateAddr(tokens.SolanaAddress)))
	}

	details := strings.Join(lines, "\n")

	inner := fmt.Sprintf(
		"%s  %s\n\n%s\n\n%s",
		lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("\u2713"),
		lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("Agent initialized successfully"),
		details,
		tagline,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorGold).
		Padding(1, 3).
		Render(inner)

	b.WriteString(box)
	return b.String()
}

func runNextStepMenu() {
	dim := ui.DimStyle
	bright := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)

	fmt.Println(dim.Render("  You'll need ") + bright.Render("Claude Desktop") + dim.Render(" or ") + bright.Render("Claude Code") + dim.Render(" installed to trade."))
	fmt.Println(dim.Render("  Download at ") + lipgloss.NewStyle().Foreground(ui.ColorBoba).Underline(true).Render("https://claude.ai/download"))
	fmt.Println()

	for {
		var next string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("What do you want to do next?").
					Options(
						huh.NewOption("Set up Claude to use Boba", "install"),
						huh.NewOption("Start trading with Claude", "launch"),
						huh.NewOption("See if everything's working", "status"),
						huh.NewOption("Back to main menu", "menu"),
					).
					Value(&next),
			),
		).WithTheme(bobaTheme())

		if err := form.Run(); err != nil || next == "menu" {
			fmt.Println()
			runInteractiveMenu()
			return
		}

		fmt.Println()

		subCmd, _, err := rootCmd.Find([]string{next})
		if err != nil || subCmd == nil || subCmd == rootCmd {
			return
		}
		if subCmd.RunE != nil {
			if err := subCmd.RunE(subCmd, []string{}); err != nil {
				fmt.Println(ui.ErrorStyle.Render(err.Error()))
			}
		} else if subCmd.Run != nil {
			subCmd.Run(subCmd, []string{})
		}

		fmt.Println()
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	ui.PrintLogo()
	fmt.Println()

	agentID := flagAgentID
	secret := flagSecret
	name := flagName

	if agentID == "" || secret == "" {
		hasCreds := true
		prompt := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Do you have your agent credentials?").
					Description("If not, we'll open the signup page for you.").
					Affirmative("Yes, let's go").
					Negative("No, sign me up").
					Value(&hasCreds),
			),
		).WithTheme(bobaTheme())

		if err := prompt.Run(); err != nil {
			return fmt.Errorf("cancelled: %w", err)
		}

		if !hasCreds {
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true).Render("  Opening https://agents.boba.xyz ..."))
			fmt.Println(ui.DimStyle.Render("  Come back and run ") + ui.BrightStyle.Render("boba login") + ui.DimStyle.Render(" once you have your credentials."))
			fmt.Println()
			openBrowser("https://agents.boba.xyz")
			return nil
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Agent Setup").
					Description("Enter your Boba Agent credentials."),
				huh.NewInput().
					Title("Agent ID").
					Description("Your unique agent identifier").
					Value(&agentID),
				huh.NewInput().
					Title("Agent Secret").
					Description("Your agent secret key").
					EchoMode(huh.EchoModePassword).
					Value(&secret),
				huh.NewInput().
					Title("Agent Name").
					Description("A friendly name for your agent (optional)").
					Value(&name),
			),
		).WithTheme(bobaTheme())

		if err := form.Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
	}

	if agentID == "" || secret == "" {
		return fmt.Errorf("agent ID and secret are required")
	}

	fmt.Println()

	var tokens *config.AuthTokens

	steps := []onboardingStep{
		{
			label: "Saving credentials to keychain...",
			fn: func() error {
				return config.SetCredentials(agentID, secret, name)
			},
		},
		{
			label: "Connecting to Boba network...",
			fn: func() error {
				// Connection is established as part of auth; small pause for UX
				time.Sleep(200 * time.Millisecond)
				return nil
			},
		},
		{
			label: "Authenticating agent...",
			fn: func() error {
				var err error
				tokens, err = auth.Authenticate()
				return err
			},
		},
		{
			label: "Registering with trading services...",
			fn: func() error {
				// Registration happens inside auth.Authenticate() already
				time.Sleep(250 * time.Millisecond)
				return nil
			},
		},
		{
			label: "Initializing wallet monitoring...",
			fn: func() error {
				// Wallet monitoring init happens inside auth.Authenticate() already
				time.Sleep(200 * time.Millisecond)
				return nil
			},
		},
	}

	model := newOnboardingModel(steps)
	model.statuses[0] = stepRunning

	p := tea.NewProgram(model, tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	m := finalModel.(onboardingModel)
	if m.failed {
		for i, e := range m.errors {
			if e != nil {
				return fmt.Errorf("%s: %w", m.steps[i].label, e)
			}
		}
		return fmt.Errorf("initialization was interrupted")
	}

	if tokens == nil {
		return fmt.Errorf("authentication failed: no tokens received")
	}

	fmt.Println(renderSuccessCard(tokens))
	fmt.Println()
	runNextStepMenu()

	return nil
}

func truncateAddr(addr string) string {
	if len(addr) >= 10 {
		return addr[:6] + "..." + addr[len(addr)-4:]
	}
	return addr
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "windows":
		_ = exec.Command("cmd", "/c", "start", url).Start()
	}
}
