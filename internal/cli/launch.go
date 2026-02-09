package cli

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Start trading with Claude",
	RunE:  runLaunch,
}

var (
	flagDesktop bool
	flagITerm   bool
)

func init() {
	launchCmd.Flags().BoolVar(&flagDesktop, "desktop", false, "Open Claude Desktop instead of Code")
	launchCmd.Flags().BoolVar(&flagITerm, "iterm", false, "Use iTerm instead of Terminal.app (macOS only)")
}

type layout int

const (
	layoutSideBySide layout = iota
	layoutStacked
	layoutProxyOnly
)

// screenBounds holds the desktop screen dimensions.
type screenBounds struct {
	x, y, w, h int
}

// windowRect describes a window position {left, top, right, bottom}.
type windowRect struct {
	left, top, right, bottom int
}

type launchStepDoneMsg struct{ idx int }
type launchStepFailMsg struct {
	idx int
	err error
}
type launchAllDoneMsg struct{}
type launchTickMsg time.Time

type launchStep struct {
	label string
	fn    func() error
}

type launchModel struct {
	selected string

	steps      []launchStep
	statuses   []stepStatus
	errors     []error
	current    int
	spinner    spinner.Model
	progress   progress.Model
	glitchTick int
	logoTick   int
	target     float64 // smooth progress target (0.0–1.0)

	width      int
	failed     bool
	done       bool
	successOut string // rendered success card for after alt-screen exit
}

func newLaunchModel(selected string, steps []launchStep) launchModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorBoba)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(44),
	)

	statuses := make([]stepStatus, len(steps))
	statuses[0] = stepRunning

	return launchModel{
		selected: selected,
		steps:    steps,
		statuses: statuses,
		errors:   make([]error, len(steps)),
		spinner:  s,
		progress: p,
		width:    60,
	}
}

func launchProgressTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return launchTickMsg(t)
	})
}

func (m launchModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runStep(0),
		launchProgressTick(),
	)
}

func (m launchModel) runStep(idx int) tea.Cmd {
	if idx >= len(m.steps) {
		return func() tea.Msg { return launchAllDoneMsg{} }
	}
	fn := m.steps[idx].fn
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)
		if err := fn(); err != nil {
			return launchStepFailMsg{idx: idx, err: err}
		}
		return launchStepDoneMsg{idx: idx}
	}
}

func (m launchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.failed = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case launchTickMsg:
		m.logoTick++
		m.glitchTick++
		// Smooth progress: inch toward target
		var cmds []tea.Cmd
		current := m.progress.Percent()
		if current < m.target {
			incr := 0.008
			if m.target-current < 0.05 {
				incr = 0.003 // slow down near target for smooth approach
			}
			cmd := m.progress.IncrPercent(incr)
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, launchProgressTick())
		return m, tea.Batch(cmds...)

	case launchStepDoneMsg:
		m.statuses[msg.idx] = stepDone
		m.glitchTick = 0
		// Bump target by 1/totalSteps
		m.target = float64(msg.idx+1) / float64(len(m.steps))
		next := msg.idx + 1
		if next < len(m.steps) {
			m.current = next
			m.statuses[next] = stepRunning
			return m, m.runStep(next)
		}
		return m, func() tea.Msg { return launchAllDoneMsg{} }

	case launchStepFailMsg:
		m.statuses[msg.idx] = stepFailed
		m.errors[msg.idx] = msg.err
		m.failed = true
		return m, tea.Quit

	case launchAllDoneMsg:
		m.done = true
		m.target = 1.0
		// Pre-render the success card before exiting alt screen
		m.successOut = m.renderSuccessCard()
		return m, tea.Quit

	case progress.FrameMsg:
		mdl, cmd := m.progress.Update(msg)
		m.progress = mdl.(progress.Model)
		return m, cmd

	default:
		m.glitchTick++
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m launchModel) View() string {
	var b strings.Builder

	logoLines := ui.LogoLines()
	gradient := ui.GradientPurple

	logoProgress := float64(m.logoTick) / 18.0
	if logoProgress > 1.0 {
		logoProgress = 1.0
	}

	glitchedLogo := ui.GlitchLines(logoLines, logoProgress)
	for i, line := range glitchedLogo {
		colorIdx := 0
		if len(gradient) > 1 && len(glitchedLogo) > 1 {
			colorIdx = i * (len(gradient) - 1) / (len(glitchedLogo) - 1)
		}
		if colorIdx >= len(gradient) {
			colorIdx = len(gradient) - 1
		}
		style := lipgloss.NewStyle().Foreground(gradient[colorIdx]).Bold(true)
		b.WriteString("  " + style.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	layoutLabel := layoutDisplayName(m.selected)
	layoutStyle := lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true)
	b.WriteString("  " + lipgloss.NewStyle().Foreground(ui.ColorDim).Render("Layout: ") + layoutStyle.Render(layoutLabel))
	b.WriteString("\n\n")

	// Steps with spinner + glitch text
	checkmark := lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("  \u2713 ")
	failmark := lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true).Render("  \u2717 ")

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
			b.WriteString(labelFailed.Render(step.label + " \u2014 FAILED"))
		default:
			scrambled := ui.GlitchText(step.label, 0.0)
			b.WriteString("    ")
			b.WriteString(labelPending.Render(scrambled))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n  ")
	b.WriteString(m.progress.View())
	b.WriteString("\n")

	borderLen := 44
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
	b.WriteString("  " + lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("\u2501", borderLen)))
	b.WriteString("\n")

	return b.String()
}

func (m launchModel) renderSuccessCard() string {
	var b strings.Builder

	b.WriteString(ui.RenderLogo())
	b.WriteString("\n\n")

	layoutLabel := layoutDisplayName(m.selected)

	inner := fmt.Sprintf(
		"%s  %s\n\n%s %s\n\n%s",
		lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("\u2713"),
		lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("Launched successfully"),
		lipgloss.NewStyle().Foreground(ui.ColorDim).Render("Layout:"),
		lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render(layoutLabel),
		lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true).Render("Happy trading!"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorGold).
		Padding(1, 3).
		Render(inner)

	b.WriteString(box)
	b.WriteString("\n\n")
	b.WriteString(ui.DimStyle.Render("  Proxy is running in a separate terminal window."))
	b.WriteString("\n")
	if m.selected != "proxy-only" {
		b.WriteString(ui.DimStyle.Render("  Claude should open momentarily."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	return b.String()
}

func parseLayout(s string) layout {
	switch s {
	case "stacked":
		return layoutStacked
	case "proxy-only":
		return layoutProxyOnly
	default:
		return layoutSideBySide
	}
}

func layoutDisplayName(s string) string {
	switch s {
	case "stacked":
		return "Stacked"
	case "proxy-only":
		return "Proxy Only"
	case "default":
		return "Default"
	default:
		return "Side by Side"
	}
}

func computeRects(l layout, s screenBounds) (proxyRect, claudeRect windowRect) {
	switch l {
	case layoutSideBySide:
		split := int(float64(s.w) * 0.65)
		proxyRect = windowRect{s.x + split, s.y, s.x + s.w, s.y + s.h}
		claudeRect = windowRect{s.x, s.y, s.x + split, s.y + s.h}
	case layoutStacked:
		split := int(float64(s.h) * 0.65)
		proxyRect = windowRect{s.x, s.y + split, s.x + s.w, s.y + s.h}
		claudeRect = windowRect{s.x, s.y, s.x + s.w, s.y + split}
	case layoutProxyOnly:
		padW := int(float64(s.w) * 0.1)
		padH := int(float64(s.h) * 0.1)
		proxyRect = windowRect{s.x + padW, s.y + padH, s.x + s.w - padW, s.y + s.h - padH}
		claudeRect = windowRect{} // unused
	}
	return
}

// getScreenBounds returns the bounds of the main screen (the one with the
// menu bar), not the full desktop span across all monitors.
func getScreenBounds() screenBounds {
	fallback := screenBounds{0, 0, 1920, 1080}

	script := `use framework "AppKit"
set scr to current application's NSScreen's mainScreen()
set f to scr's visibleFrame()
set x to (f's origin's x) as integer
set y to (f's origin's y) as integer
set w to (f's |size|'s width) as integer
set h to (f's |size|'s height) as integer
return (x as text) & "," & (y as text) & "," & (w as text) & "," & (h as text)`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return fallback
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 4 {
		return fallback
	}

	vals := make([]int, 4)
	for i, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return fallback
		}
		vals[i] = v
	}

	return screenBounds{
		x: vals[0],
		y: vals[1],
		w: vals[2],
		h: vals[3],
	}
}

// escapeAppleScript escapes a string for safe interpolation into AppleScript.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// launchTerminalWindow opens a macOS terminal window at the given position.
func launchTerminalWindow(cmd string, rect windowRect, useITerm bool) error {
	escaped := escapeAppleScript(cmd)
	var script string

	if useITerm {
		script = fmt.Sprintf(`tell application "iTerm"
	activate
	set newWindow to (create window with default profile)
	set bounds of newWindow to {%d, %d, %d, %d}
	tell current session of newWindow
		write text "%s"
	end tell
end tell`, rect.left, rect.top, rect.right, rect.bottom, escaped)
	} else {
		script = fmt.Sprintf(`tell application "Terminal"
	activate
	do script "%s"
	delay 0.5
	set bounds of front window to {%d, %d, %d, %d}
end tell`, escaped, rect.left, rect.top, rect.right, rect.bottom)
	}

	return exec.Command("osascript", "-e", script).Run()
}

// launchTerminalLinux tries common terminal emulators in order.
func launchTerminalLinux(bobaPath string) error {
	startCmd := bobaPath + " start"

	terminals := []struct {
		bin  string
		args func(string) []string
	}{
		{"gnome-terminal", func(cmd string) []string { return []string{"--", "bash", "-c", cmd} }},
		{"konsole", func(cmd string) []string { return []string{"-e", "bash", "-c", cmd} }},
		{"xfce4-terminal", func(cmd string) []string { return []string{"-e", "bash -c '" + cmd + "'"} }},
		{"xterm", func(cmd string) []string { return []string{"-e", "bash", "-c", cmd} }},
	}

	for _, t := range terminals {
		if binPath, err := exec.LookPath(t.bin); err == nil {
			c := exec.Command(binPath, t.args(startCmd)...)
			return c.Start()
		}
	}
	return fmt.Errorf("no supported terminal emulator found (tried gnome-terminal, konsole, xfce4-terminal, xterm)")
}

// launchTerminalWindows tries Windows Terminal, then falls back to cmd.exe.
func launchTerminalWindows(bobaPath string) error {
	startCmd := bobaPath + " start"

	if wtPath, err := exec.LookPath("wt.exe"); err == nil {
		c := exec.Command(wtPath, "new-tab", "--", startCmd)
		return c.Start()
	}
	c := exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k", startCmd)
	return c.Start()
}

// openClaudeDesktop attempts to open the Claude Desktop application.
func openClaudeDesktop() error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-a", "Claude").Run()
	case "linux":
		if binPath, err := exec.LookPath("claude-desktop"); err == nil {
			c := exec.Command(binPath)
			return c.Start()
		}
		c := exec.Command("xdg-open", "claude://")
		return c.Start()
	case "windows":
		c := exec.Command("cmd.exe", "/c", "start", "", "claude://")
		return c.Start()
	}
	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

func waitForHealth(port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d/health", port)
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("proxy did not become healthy within %s", timeout)
}

func runLaunch(cmd *cobra.Command, args []string) error {
	if !config.HasCredentials() {
		return fmt.Errorf("no credentials configured. Run 'boba login' first")
	}

	bobaPath, _ := os.Executable()
	bobaPath, _ = filepath.EvalSymlinks(bobaPath)
	port := config.GetProxyPort()

	if runtime.GOOS == "darwin" {
		return runLaunchMacOS(bobaPath, port)
	}
	return runLaunchGeneric(bobaPath, port)
}

func runLaunchMacOS(bobaPath string, port int) error {
	ui.PrintLogo()
	fmt.Println()

	selected := "side-by-side"
	claudeApp := "code"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which Claude do you use?").
				Options(
					huh.NewOption("Claude Code (runs in terminal)", "code"),
					huh.NewOption("Claude Desktop (standalone app)", "desktop"),
					huh.NewOption("Neither — just run the proxy", "none"),
				).
				Value(&claudeApp),
			huh.NewSelect[string]().
				Title("How should windows be arranged?").
				Options(
					huh.NewOption("Side by Side", "side-by-side"),
					huh.NewOption("Stacked", "stacked"),
					huh.NewOption("Proxy Only", "proxy-only"),
				).
				Value(&selected),
		),
	).WithTheme(ui.BobaTheme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("selection cancelled")
	}

	if claudeApp == "none" {
		selected = "proxy-only"
	}

	fmt.Println()

	chosenLayout := parseLayout(selected)
	bounds := getScreenBounds()
	cwd, _ := os.Getwd()

	proxyRect, claudeRect := computeRects(chosenLayout, bounds)

	steps := []launchStep{
		{
			label: "Initializing proxy...",
			fn: func() error {
				return launchTerminalWindow(bobaPath+" start", proxyRect, flagITerm)
			},
		},
		{
			label: "Waiting for proxy...",
			fn: func() error {
				return waitForHealth(port, 15*time.Second)
			},
		},
	}

	if chosenLayout != layoutProxyOnly {
		if claudeApp == "desktop" || flagDesktop {
			steps = append(steps, launchStep{
				label: "Opening Claude Desktop...",
				fn: func() error {
					return openClaudeDesktop()
				},
			})
		} else {
			steps = append(steps, launchStep{
				label: "Opening Claude Code...",
				fn: func() error {
					shellCmd := fmt.Sprintf("cd '%s' && claude", escapeAppleScript(cwd))
					return launchTerminalWindow(shellCmd, claudeRect, flagITerm)
				},
			})
		}
	}

	return runLaunchAnimation(selected, steps)
}

func runLaunchGeneric(bobaPath string, port int) error {
	ui.PrintLogo()
	fmt.Println()

	claudeApp := "code"
	if flagDesktop {
		claudeApp = "desktop"
	}

	if !flagDesktop {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Which Claude do you use?").
					Options(
						huh.NewOption("Claude Code (runs in terminal)", "code"),
						huh.NewOption("Claude Desktop (standalone app)", "desktop"),
						huh.NewOption("Neither — just run the proxy", "none"),
					).
					Value(&claudeApp),
			),
		).WithTheme(ui.BobaTheme())

		if err := form.Run(); err != nil {
			return fmt.Errorf("selection cancelled")
		}
		fmt.Println()
	}

	selected := "default"

	steps := []launchStep{
		{
			label: "Initializing proxy...",
			fn: func() error {
				switch runtime.GOOS {
				case "linux":
					return launchTerminalLinux(bobaPath)
				case "windows":
					return launchTerminalWindows(bobaPath)
				default:
					return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
				}
			},
		},
		{
			label: "Waiting for proxy...",
			fn: func() error {
				return waitForHealth(port, 15*time.Second)
			},
		},
	}

	if claudeApp == "desktop" {
		steps = append(steps, launchStep{
			label: "Opening Claude Desktop...",
			fn: func() error {
				return openClaudeDesktop()
			},
		})
	}

	return runLaunchAnimation(selected, steps)
}

func runLaunchAnimation(selected string, steps []launchStep) error {
	model := newLaunchModel(selected, steps)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("launch failed: %w", err)
	}

	m := finalModel.(launchModel)
	if m.failed {
		for i, e := range m.errors {
			if e != nil {
				return fmt.Errorf("%s: %w", m.steps[i].label, e)
			}
		}
		return fmt.Errorf("launch was cancelled")
	}

	// Print success card to normal terminal (persists after alt screen exit)
	fmt.Print(m.successOut)

	return nil
}
