package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ASCII art logo for BOBA AGENTS.
var logoLines = []string{
	`        ██████╗  ██████╗ ██████╗  █████╗        `,
	`        ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗       `,
	`        ██████╔╝██║   ██║██████╔╝███████║       `,
	`        ██╔══██╗██║   ██║██╔══██╗██╔══██║       `,
	`        ██████╔╝╚██████╔╝██████╔╝██║  ██║       `,
	`        ╚═════╝  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝       `,
	`                             A G E N T S       `,
}

// Full gradient from deep purple through boba to bright lavender.
var logoGradient = []lipgloss.Color{
	lipgloss.Color("#6B3FA0"),
	lipgloss.Color("#7B52B5"),
	lipgloss.Color("#8A5FD1"),
	lipgloss.Color("#9B72E0"),
	lipgloss.Color("#B184F5"),
	lipgloss.Color("#C098FF"),
	lipgloss.Color("#D4A5FF"),
}

// PrintLogo prints the BOBA AGENTS ASCII logo with a purple gradient.
func PrintLogo() {
	fmt.Println(RenderLogo())
}

// RenderLogo returns the colored logo as a string.
func RenderLogo() string {
	var b strings.Builder
	for i, line := range logoLines {
		color := ColorBoba
		if i < len(logoGradient) {
			color = logoGradient[i]
		}
		style := lipgloss.NewStyle().Foreground(color).Bold(true)
		b.WriteString(style.Render(line))
		if i < len(logoLines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// RenderLogoCompact returns a smaller tagline version.
func RenderLogoCompact() string {
	tag := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(ColorBoba).
		Bold(true).
		Padding(0, 1).
		Render("BOBA")
	agents := lipgloss.NewStyle().
		Foreground(ColorBright).
		Bold(true).
		Render(" AGENTS")
	return tag + agents
}

