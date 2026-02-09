package formatter

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatPortfolio renders a portfolio summary with total value, P&L, and a
// holdings table. Handles the actual MCP response format:
//
//	{
//	  "total_value_usd": "2150.50",
//	  "position_value_usd": "1275.50",
//	  "native_value_usd": "875.00",
//	  "position_count": 3,
//	  "positions": [ { "symbol": "POPCAT", "value_usd": "250.00", ... } ],
//	  "native_balances": [ { "symbol": "SOL", "balance_usd": "875.00", ... } ]
//	}
func FormatPortfolio(data map[string]any) string {
	totalValue := getFloat(data, "total_value_usd")
	positionValue := getFloat(data, "position_value_usd")
	nativeValue := getFloat(data, "native_value_usd")

	// Header
	header := lipgloss.NewStyle().
		Foreground(ui.ColorGold).
		Bold(true).
		Render("PORTFOLIO")

	// Total value
	totalLine := lipgloss.NewStyle().
		Foreground(ui.ColorGold).
		Bold(true).
		Render(fmt.Sprintf("Total Value: %s", FormatUSD(totalValue)))

	// Breakdown
	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(14)
	breakdownLines := []string{}
	if positionValue > 0 {
		breakdownLines = append(breakdownLines, labelStyle.Render("Positions")+FormatUSD(positionValue))
	}
	if nativeValue > 0 {
		breakdownLines = append(breakdownLines, labelStyle.Render("Native")+FormatUSD(nativeValue))
	}

	// Positions table — try "positions" first, fall back to "tokens"
	positions, _ := data["positions"].([]any)
	if positions == nil {
		positions, _ = data["tokens"].([]any)
	}

	// Sort positions by value_usd descending (largest first)
	sort.Slice(positions, func(i, j int) bool {
		pi, _ := positions[i].(map[string]any)
		pj, _ := positions[j].(map[string]any)
		vi := getFloat(pi, "value_usd")
		vj := getFloat(pj, "value_usd")
		return vi > vj
	})

	// Native balances
	nativeBalances, _ := data["native_balances"].([]any)

	maxRows := 8
	allPositions := positions
	showMore := len(positions) > maxRows
	if len(positions) > maxRows {
		positions = positions[:maxRows]
	}

	var rows []string
	if len(positions) > 0 {
		compact := isCompact()

		var colSym, colVal, colAlloc, colPrice, colPnl int
		if compact {
			colSym = 8
			colVal = 12
			colPnl = 10
		} else {
			colSym = 10
			colVal = 14
			colAlloc = 14
			colPrice = 14
			colPnl = 12
		}

		// Table header
		headerParts := []string{
			lipgloss.NewStyle().Width(colSym).Bold(true).Render("Symbol"),
			lipgloss.NewStyle().Width(colVal).Bold(true).Render("Value"),
		}
		if !compact {
			headerParts = append(headerParts,
				lipgloss.NewStyle().Width(colAlloc).Bold(true).Render("Allocation"),
				lipgloss.NewStyle().Width(colPrice).Bold(true).Render("Price"),
			)
		}
		headerParts = append(headerParts, lipgloss.NewStyle().Width(colPnl).Bold(true).Render("PnL"))
		headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)
		rows = append(rows, headerRow)

		totalCols := colSym + colVal + colPnl
		if !compact {
			totalCols += colAlloc + colPrice
		}
		rows = append(rows, sepLine(totalCols))

		for _, t := range positions {
			token, ok := t.(map[string]any)
			if !ok {
				continue
			}

			symbol := getString(token, "symbol")
			if symbol == "" {
				symbol = getString(token, "name")
			}
			if symbol == "" {
				symbol = getString(token, "token_symbol")
			}

			value := getFloat(token, "value_usd")
			if value == 0 {
				value = getFloat(token, "usd_value")
			}
			if value == 0 {
				value = getFloat(token, "balance_usd")
			}

			price := getFloat(token, "price_usd")
			if price == 0 {
				price = getFloat(token, "price")
			}

			pnlPct := getFloat(token, "pnl_percent")
			if pnlPct == 0 {
				pnlPct = getFloat(token, "roi_percent")
			}

			allocation := 0.0
			if totalValue > 0 {
				allocation = value / totalValue
			}

			rowParts := []string{
				lipgloss.NewStyle().Width(colSym).Foreground(ui.ColorBright).Render(symbol),
				lipgloss.NewStyle().Width(colVal).Render(FormatUSD(value)),
			}
			if !compact {
				rowParts = append(rowParts,
					lipgloss.NewStyle().Width(colAlloc).Render(ProgressBar(allocation, 1.0, 10)),
					lipgloss.NewStyle().Width(colPrice).Render(FormatUSD(price)),
				)
			}
			rowParts = append(rowParts, lipgloss.NewStyle().Width(colPnl).Render(FormatPercent(pnlPct)))
			row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
			rows = append(rows, row)
		}

		if showMore {
			remaining := len(allPositions) - maxRows
			rows = append(rows, ui.DimStyle.Render(fmt.Sprintf("...and %d more positions", remaining)))
		}
	}

	// Native balances section
	if len(nativeBalances) > 0 {
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Native Balances"))
		for _, nb := range nativeBalances {
			bal, ok := nb.(map[string]any)
			if !ok {
				continue
			}
			symbol := getString(bal, "symbol")
			balance := getFloat(bal, "balance")
			balUSD := getFloat(bal, "balance_usd")
			chainName := getString(bal, "chain_name")
			chain := ""
			if chainName != "" {
				chain = " (" + chainName + ")"
			}
			row := fmt.Sprintf("  %s%s  %s  %s",
				lipgloss.NewStyle().Foreground(ui.ColorBright).Render(symbol),
				ui.DimStyle.Render(chain),
				FormatNumber(balance),
				FormatUSD(balUSD),
			)
			rows = append(rows, row)
		}
	}

	holdingsTable := strings.Join(rows, "\n")

	// Compose the full output
	sections := []string{header, "", totalLine}
	if len(breakdownLines) > 0 {
		sections = append(sections, strings.Join(breakdownLines, "\n"))
	}
	if holdingsTable != "" {
		sections = append(sections, "", holdingsTable)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return ui.GoldBoxBorder.Render(content)
}

// getFloat safely extracts a float64 from a map with a string key.
// Handles float64, int, int64, and string representations of numbers.
func getFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		s := strings.TrimSpace(n)
		s = strings.TrimSuffix(s, "%")
		s = strings.ReplaceAll(s, ",", "")
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

// getString safely extracts a string from a map with a string key.
func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// getBool safely extracts a bool from a map with a string key.
func getBool(m map[string]any, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}
