package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatTrendingTokens renders a ranked list of trending tokens with medals
// for the top three and price/change data for each.
func FormatTrendingTokens(data map[string]any) string {
	tokens, _ := data["tokens"].([]any)
	if tokens == nil {
		tokens, _ = data["results"].([]any)
	}
	if len(tokens) == 0 {
		return ui.DimStyle.Render("No trending tokens available.")
	}

	header := lipgloss.NewStyle().
		Foreground(ui.ColorGold).
		Bold(true).
		Render("TRENDING \U0001F525")

	maxTokens := 10
	if len(tokens) > maxTokens {
		tokens = tokens[:maxTokens]
	}

	var rows []string
	for i, t := range tokens {
		token, ok := t.(map[string]any)
		if !ok {
			continue
		}

		rank := i + 1
		medal := rankMedal(rank)

		symbol := getString(token, "symbol")
		price := getFloat(token, "price_usd")
		if price == 0 {
			price = getFloat(token, "price")
		}
		change24h := getFloat(token, "price_change_24h")
		if change24h == 0 {
			change24h = getFloat(token, "change_24h")
		}

		symbolStyle := lipgloss.NewStyle().
			Foreground(ui.ColorBright).
			Bold(true).
			Width(10)

		row := fmt.Sprintf("%s  %s  %s  %s",
			medal,
			symbolStyle.Render(symbol),
			lipgloss.NewStyle().Width(14).Render(FormatUSD(price)),
			FormatPercent(change24h),
		)
		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// rankMedal returns a medal emoji for ranks 1-3 or a formatted number for
// other ranks.
func rankMedal(rank int) string {
	switch rank {
	case 1:
		return "\U0001F947" // gold medal
	case 2:
		return "\U0001F948" // silver medal
	case 3:
		return "\U0001F949" // bronze medal
	default:
		return fmt.Sprintf("%-2d", rank)
	}
}
