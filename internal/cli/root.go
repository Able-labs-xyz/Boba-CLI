package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/logger"
	"github.com/tradeboba/boba-cli/internal/ui"
	"github.com/tradeboba/boba-cli/internal/version"
)

type menuOption struct {
	label string
	value string
}

func buildMenuOptions() []menuOption {
	hasCreds := config.HasCredentials()

	options := []menuOption{
		{"login     Log in with your agent credentials", "login"},
		{"install   Set up Claude to use Boba", "install"},
	}

	if hasCreds {
		options = append(options,
			menuOption{"launch    Start trading with Claude", "launch"},
			menuOption{"start     Run the Boba proxy", "start"},
			menuOption{"status    See if everything's working", "status"},
		)
	}

	options = append(options,
		menuOption{"config    Change your settings", "config"},
		menuOption{"auth      Test your connection", "auth"},
	)

	if hasCreds {
		options = append(options,
			menuOption{"logout    Sign out", "logout"},
		)
	}

	return options
}

var rootCmd = &cobra.Command{
	Use:   "boba",
	Short: "Boba Agent CLI — Connect AI agents to Boba trading",
	Long: lipgloss.NewStyle().Foreground(ui.ColorBoba).Render(
		"Boba Agent CLI — Connect AI agents to decentralized trading via the Boba MCP protocol"),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Load()
		logger.Init(config.GetLogLevel())
		ensureMCPConfig()
	},
	Version: version.Version,
}

func runInteractiveMenu() {
	options := buildMenuOptions()

	m := newMenuModel(options)
	p := tea.NewProgram(m, tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		os.Exit(1)
	}

	result := finalModel.(menuModel)
	if result.chosen == "" {
		os.Exit(0)
	}

	fmt.Println()

	subCmd, _, err := rootCmd.Find([]string{result.chosen})
	if err != nil || subCmd == nil || subCmd == rootCmd {
		fmt.Println(ui.ErrorStyle.Render("Unknown command: " + result.chosen))
		os.Exit(1)
	}

	if subCmd.RunE != nil {
		if err := subCmd.RunE(subCmd, []string{}); err != nil {
			fmt.Println(ui.ErrorStyle.Render(err.Error()))
			os.Exit(1)
		}
	} else if subCmd.Run != nil {
		subCmd.Run(subCmd, []string{})
	}
}

type menuPhase int

const (
	menuPhaseAnimation menuPhase = iota
	menuPhaseSelect
)

type menuTickMsg struct{}

type menuModel struct {
	phase    menuPhase
	frame    int
	tagline  string
	items    []menuOption
	cursor int
	chosen string
}

func newMenuModel(items []menuOption) menuModel {
	return menuModel{
		tagline: "Connect AI agents to decentralized trading",
		items:   items,
	}
}

func menuTick() tea.Cmd {
	return tea.Tick(30*time.Millisecond, func(_ time.Time) tea.Msg {
		return menuTickMsg{}
	})
}

func (m menuModel) Init() tea.Cmd {
	return menuTick()
}

const (
	logoFrames   = 12 // logo decrypts over frames 0-12
	dividerFrame = 13
	taglineStart = 14
	taglineSpeed = 4 // chars per frame
	itemStagger  = 2 // frames between each item appearing
	itemDecrypt  = 6 // frames for each item to fully decrypt
)

func (m menuModel) taglineDoneFrame() int {
	return taglineStart + (len([]rune(m.tagline))+taglineSpeed-1)/taglineSpeed
}

func (m menuModel) menuStartFrame() int {
	return m.taglineDoneFrame() + 2
}

func (m menuModel) animDoneFrame() int {
	msf := m.menuStartFrame()
	lastItemStart := msf + (len(m.items)-1)*itemStagger
	return lastItemStart + itemDecrypt
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		if m.phase == menuPhaseAnimation {
			m.phase = menuPhaseSelect
			m.frame = m.animDoneFrame()
			return m, nil
		}

		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.items[m.cursor].value
			return m, tea.Quit
		}
		return m, nil

	case menuTickMsg:
		if m.phase == menuPhaseAnimation {
			m.frame++
			if m.frame >= m.animDoneFrame()+3 {
				m.phase = menuPhaseSelect
				return m, nil
			}
			return m, menuTick()
		}
		return m, nil
	}
	return m, nil
}

func (m menuModel) View() string {
	var b strings.Builder

	b.WriteString("\n\n")

	logoLines := ui.LogoLines()
	gradient := ui.GradientPurple

	logoProgress := 1.0
	if m.phase == menuPhaseAnimation {
		logoProgress = float64(m.frame) / float64(logoFrames)
		if logoProgress > 1.0 {
			logoProgress = 1.0
		}
	}

	glitched := ui.GlitchLines(logoLines, logoProgress)
	for i, line := range glitched {
		colorIdx := 0
		if len(gradient) > 1 && len(glitched) > 1 {
			colorIdx = i * (len(gradient) - 1) / (len(glitched) - 1)
		}
		if colorIdx >= len(gradient) {
			colorIdx = len(gradient) - 1
		}
		style := lipgloss.NewStyle().Foreground(gradient[colorIdx]).Bold(true)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	showDivider := m.phase == menuPhaseSelect || m.frame >= dividerFrame
	if showDivider {
		divStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
		b.WriteString("\n  " + divStyle.Render(strings.Repeat("\u2501", 46)))
	} else {
		b.WriteString("\n")
	}
	b.WriteString("\n")

	tagRunes := []rune(m.tagline)
	if m.phase == menuPhaseSelect {
		tagStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Italic(true)
		b.WriteString("  " + tagStyle.Render(m.tagline))
	} else if m.frame >= taglineStart {
		charsRevealed := (m.frame - taglineStart) * taglineSpeed
		if charsRevealed >= len(tagRunes) {
			tagStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Italic(true)
			b.WriteString("  " + tagStyle.Render(m.tagline))
		} else {
			visible := string(tagRunes[:charsRevealed])
			cursor := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true).Render("\u2588")
			tagStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Italic(true)
			b.WriteString("  " + tagStyle.Render(visible) + cursor)
		}
	}
	b.WriteString("\n\n")

	if m.phase == menuPhaseSelect {
		m.renderSelectItems(&b)
	} else {
		m.renderAnimItems(&b)
	}

	b.WriteString("\n")

	b.WriteString("\n")
	if m.phase == menuPhaseSelect {
		borderStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
		b.WriteString("  " + borderStyle.Render(strings.Repeat("\u2501", 46)))
		b.WriteString("\n")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
		b.WriteString("  " + hintStyle.Render("\u2191\u2193 navigate  enter select  q quit"))
	} else {
		var borderColor lipgloss.Color
		if m.frame%4 < 2 {
			borderColor = ui.ColorDim
		} else {
			borderColor = ui.ColorBoba
		}
		b.WriteString("  " + lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("\u2501", 46)))
	}
	b.WriteString("\n")

	return b.String()
}

// renderAnimItems renders menu items with staggered glitch-decrypt.
func (m menuModel) renderAnimItems(b *strings.Builder) {
	msf := m.menuStartFrame()

	cmdStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)
	glitchStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)

	for i, opt := range m.items {
		itemStart := msf + i*itemStagger
		if m.frame < itemStart {
			b.WriteString("\n")
			continue
		}

		elapsed := m.frame - itemStart
		p := float64(elapsed) / float64(itemDecrypt)
		if p > 1.0 {
			p = 1.0
		}

		if p >= 1.0 {
			b.WriteString("    " + cmdStyle.Render(opt.label) + "\n")
		} else {
			glitchedLabel := ui.GlitchText(opt.label, p)
			b.WriteString("    " + glitchStyle.Render(glitchedLabel) + "\n")
		}
	}
}

// renderSelectItems renders the interactive select list with cursor.
func (m menuModel) renderSelectItems(b *strings.Builder) {
	selectedStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
	cursorStyle := lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true)

	for i, opt := range m.items {
		if i == m.cursor {
			b.WriteString("  " + cursorStyle.Render("\u25b8 ") + selectedStyle.Render(opt.label))
		} else {
			b.WriteString("    " + normalStyle.Render(opt.label))
		}
		b.WriteString("\n")
	}
}


func init() {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runInteractiveMenu()
	}

	rootCmd.SetVersionTemplate(
		lipgloss.NewStyle().Foreground(ui.ColorBoba).Render("boba") +
			" version " +
			lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true).Render("{{.Version}}") +
			"\n",
	)

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(mcpCmd)
}

// ensureMCPConfig silently updates the MCP config so Claude always
// points to the current boba binary, even after npm updates.
func ensureMCPConfig() {
	bobaPath, err := exec.LookPath("boba")
	if err != nil {
		return
	}
	bobaPath, _ = filepath.Abs(bobaPath)

	var mcpCommand string
	var mcpArgs []string
	if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(bobaPath), ".cmd") {
		mcpCommand = "cmd.exe"
		mcpArgs = []string{"/c", bobaPath, "mcp"}
	} else {
		mcpCommand = bobaPath
		mcpArgs = []string{"mcp"}
	}

	_ = installDesktop(mcpCommand, mcpArgs)
	_ = installCode(mcpCommand, mcpArgs)
}

func Execute() error {
	return rootCmd.Execute()
}
