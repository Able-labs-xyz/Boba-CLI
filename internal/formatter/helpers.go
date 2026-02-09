package formatter

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// TermWidth is the terminal width used by formatters for responsive tables.
// Set by the TUI on init and resize. Default 80.
var TermWidth = 80

// contentWidth returns the usable width for table content inside a box border.
// Box border uses 2 chars each side for border + 2 chars each side for padding = 8 total.
// Plus 4 chars indent from activity log indentation.
func contentWidth() int {
	w := TermWidth - 12 // borders + padding + indent
	if w < 30 {
		w = 30
	}
	return w
}

// sepLine returns a dim separator line capped to content width.
func sepLine(cols int) string {
	w := cols
	cw := contentWidth()
	if w > cw {
		w = cw
	}
	return ui.DimStyle.Render(strings.Repeat("─", w))
}

// isCompact returns true when the terminal is too narrow for full tables.
func isCompact() bool {
	return TermWidth < 90
}

// FormatUSD formats a float64 value as a USD currency string with appropriate
// suffix (B, M, K) and precision, styled in gold.
func FormatUSD(value float64) string {
	style := lipgloss.NewStyle().Foreground(ui.ColorGold)

	var formatted string
	abs := math.Abs(value)
	sign := ""
	if value < 0 {
		sign = "-"
	}

	switch {
	case abs >= 1_000_000_000:
		formatted = fmt.Sprintf("%s$%.1fB", sign, abs/1_000_000_000)
	case abs >= 1_000_000:
		formatted = fmt.Sprintf("%s$%.1fM", sign, abs/1_000_000)
	case abs >= 1_000:
		formatted = fmt.Sprintf("%s$%.1fK", sign, abs/1_000)
	case abs >= 1:
		formatted = fmt.Sprintf("%s$%.2f", sign, abs)
	case abs >= 0.01:
		formatted = fmt.Sprintf("%s$%.4f", sign, abs)
	default:
		formatted = fmt.Sprintf("%s$%.8f", sign, abs)
	}

	return style.Render(formatted)
}

// FormatPercent formats a float64 as a percentage with color and direction
// indicator. Positive values are green with an up arrow, negative values are
// red with a down arrow, and zero is rendered dimly.
func FormatPercent(value float64) string {
	switch {
	case value > 0:
		style := lipgloss.NewStyle().Foreground(ui.ColorGreen)
		return style.Render(fmt.Sprintf("▲ %.2f%%", value))
	case value < 0:
		style := lipgloss.NewStyle().Foreground(ui.ColorRed)
		return style.Render(fmt.Sprintf("▼ %.2f%%", value))
	default:
		return ui.DimStyle.Render("0.00%")
	}
}

// FormatNumber formats a large number with B/M/K suffixes for readability.
func FormatNumber(value float64) string {
	abs := math.Abs(value)
	sign := ""
	if value < 0 {
		sign = "-"
	}

	switch {
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%s%.1fB", sign, abs/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("%s%.1fM", sign, abs/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%s%.1fK", sign, abs/1_000)
	default:
		return fmt.Sprintf("%s%.2f", sign, abs)
	}
}

// TruncateAddress shortens a blockchain address by keeping the first 6 and
// last 4 characters with "..." in between.
func TruncateAddress(addr string) string {
	if len(addr) >= 10 {
		return addr[:6] + "..." + addr[len(addr)-4:]
	}
	return addr
}

// Sparkline renders a sparkline string from a slice of float64 values using
// Unicode block characters. Values are normalized to the min/max range.
func Sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	spread := maxVal - minVal
	if spread == 0 {
		// All values are the same; use the middle block.
		return strings.Repeat(string(blocks[3]), len(values))
	}

	var b strings.Builder
	for _, v := range values {
		normalized := (v - minVal) / spread
		idx := int(normalized * 7)
		if idx > 7 {
			idx = 7
		}
		b.WriteRune(blocks[idx])
	}

	return b.String()
}

// ProgressBar renders a horizontal progress bar of the given width using filled
// and empty block characters. The filled portion is colored with the boba color.
func ProgressBar(current, total float64, width int) string {
	if total <= 0 || width <= 0 {
		return strings.Repeat("░", width)
	}

	ratio := current / total
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	filled := int(math.Round(ratio * float64(width)))
	empty := width - filled

	filledStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba)
	filledStr := filledStyle.Render(strings.Repeat("█", filled))
	emptyStr := strings.Repeat("░", empty)

	return filledStr + emptyStr
}
