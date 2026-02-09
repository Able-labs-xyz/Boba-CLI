package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// BobaTheme returns a Huh form theme styled with the Boba brand colors.
func BobaTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Purple palette
	purple := lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: string(ColorBoba)}
	brightPurple := lipgloss.AdaptiveColor{Light: "#9B59B6", Dark: string(ColorBright)}
	dimPurple := lipgloss.AdaptiveColor{Light: "#6C3483", Dark: string(ColorDim)}
	gold := lipgloss.AdaptiveColor{Light: "#B8860B", Dark: string(ColorGold)}

	// Focused field styling
	t.Focused.Base = t.Focused.Base.BorderForeground(purple)
	t.Focused.Title = t.Focused.Title.Foreground(brightPurple).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(dimPurple)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(gold)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(purple)

	// Select styling
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(gold)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: string(ColorBright)})
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(dimPurple)

	// Blurred field styling
	t.Blurred.Base = t.Blurred.Base.BorderForeground(dimPurple)
	t.Blurred.Title = t.Blurred.Title.Foreground(dimPurple)
	t.Blurred.Description = t.Blurred.Description.Foreground(dimPurple)
	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(dimPurple)

	return t
}
