package cli

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

type revealTickMsg struct{}

type revealModel struct {
	lines []string
	frame int
	done  bool
}

const revealFrames = 25 // ~0.8s at 33ms/tick

func revealTick() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(_ time.Time) tea.Msg { return revealTickMsg{} })
}

func (m revealModel) Init() tea.Cmd {
	return revealTick()
}

func (m revealModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.done = true
		return m, tea.Quit
	case revealTickMsg:
		m.frame++
		if m.frame >= revealFrames+3 {
			m.done = true
			return m, tea.Quit
		}
		return m, revealTick()
	}
	return m, nil
}

// lineProgress returns 0.0–1.0 for how decrypted a given line is.
// Creates a top-to-bottom wave where upper lines resolve first.
func (m revealModel) lineProgress(idx int) float64 {
	global := float64(m.frame) / float64(revealFrames)
	if global > 1.0 {
		global = 1.0
	}
	n := float64(len(m.lines))
	if n == 0 {
		return 1.0
	}
	p := global*1.3 - float64(idx)/n*0.3
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func (m revealModel) View() string {
	var b strings.Builder

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

	for i, line := range m.lines {
		p := m.lineProgress(i)
		if p >= 1.0 || m.done {
			b.WriteString(line)
		} else {
			plain := stripAnsi(line)
			if strings.TrimSpace(plain) == "" {
				// Empty / whitespace lines — show as-is
				b.WriteString(line)
			} else {
				glitched := ui.GlitchText(plain, p)
				b.WriteString(dimStyle.Render(glitched))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// runScanReveal runs the glitch-decrypt reveal animation for pre-rendered lines.
func runScanReveal(lines []string) {
	model := revealModel{lines: lines}
	p := tea.NewProgram(model, tea.WithInputTTY())
	if _, err := p.Run(); err != nil {
		for _, l := range lines {
			fmt.Println(l)
		}
	}
}
