package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatTokenSearch renders a table of token search results with columns for
// symbol, price, market cap, volume, and 24h change.
// Response: { "tokens": [{ "symbol", "price_usd", "market_cap", "volume_24h", "price_change_24h", ... }] }
func FormatTokenSearch(data map[string]any) string {
	tokens, _ := data["tokens"].([]any)
	if tokens == nil {
		tokens, _ = data["results"].([]any)
	}
	if len(tokens) == 0 {
		return ui.DimStyle.Render("No tokens found.")
	}

	maxRows := 15
	if len(tokens) > maxRows {
		tokens = tokens[:maxRows]
	}

	compact := isCompact()

	var colSymbol, colPrice, colMcap, colVol, colChange int
	if compact {
		colSymbol = 10
		colPrice = 12
		colMcap = 12
		colChange = 10
	} else {
		colSymbol = 12
		colPrice = 16
		colMcap = 14
		colVol = 14
		colChange = 12
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

	// Table header
	headerParts := []string{
		headerStyle.Width(colSymbol).Render("Symbol"),
		headerStyle.Width(colPrice).Render("Price"),
		headerStyle.Width(colMcap).Render("Mkt Cap"),
	}
	if !compact {
		headerParts = append(headerParts, headerStyle.Width(colVol).Render("Vol 24h"))
	}
	headerParts = append(headerParts, headerStyle.Width(colChange).Render("24h"))
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := colSymbol + colPrice + colMcap + colChange
	if !compact {
		totalCols += colVol
	}

	var rows []string
	rows = append(rows, header)
	rows = append(rows, sepLine(totalCols))

	for _, t := range tokens {
		token, ok := t.(map[string]any)
		if !ok {
			continue
		}

		symbol := getString(token, "symbol")
		price := getFloat(token, "price_usd")
		if price == 0 {
			price = getFloat(token, "price")
		}
		mcap := getFloat(token, "market_cap")
		vol := getFloat(token, "volume_24h")
		change := getFloat(token, "price_change_24h")

		rowParts := []string{
			lipgloss.NewStyle().Width(colSymbol).Foreground(ui.ColorBright).Render(symbol),
			lipgloss.NewStyle().Width(colPrice).Render(FormatUSD(price)),
			lipgloss.NewStyle().Width(colMcap).Render(FormatUSD(mcap)),
		}
		if !compact {
			rowParts = append(rowParts, lipgloss.NewStyle().Width(colVol).Render(FormatUSD(vol)))
		}
		_ = vol // used in full mode
		rowParts = append(rowParts, lipgloss.NewStyle().Width(colChange).Render(FormatPercent(change)))
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	title := ui.TitleStyle.Render("TOKEN SEARCH RESULTS")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// FormatTokenInfo renders detailed information about a single token including
// stats, address, price changes, and optional security audit data.
// Response: { "name", "symbol", "price_usd", "market_cap", "liquidity",
//   "volume_24h", "holders", "address", "chain_id", "launchpad",
//   "price_change_5m", "price_change_1h", "price_change_4h", "price_change_24h" }
func FormatTokenInfo(data map[string]any) string {
	name := getString(data, "name")
	symbol := getString(data, "symbol")
	price := getFloat(data, "price_usd")
	if price == 0 {
		price = getFloat(data, "price")
	}
	mcap := getFloat(data, "market_cap")
	vol := getFloat(data, "volume_24h")
	liq := getFloat(data, "liquidity")
	holders := getFloat(data, "holders")
	address := getString(data, "address")
	chainID := getString(data, "chain_id")
	launchpad := getString(data, "launchpad")

	// Header with token name and symbol
	header := lipgloss.NewStyle().
		Foreground(ui.ColorBoba).
		Bold(true).
		Render(fmt.Sprintf("%s (%s)", name, symbol))

	// Stats section
	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(14)
	var stats []string
	stats = append(stats, labelStyle.Render("Price")+smartFormatPrice(price))
	stats = append(stats, labelStyle.Render("Market Cap")+FormatUSD(mcap))
	stats = append(stats, labelStyle.Render("Volume 24h")+FormatUSD(vol))
	stats = append(stats, labelStyle.Render("Liquidity")+FormatUSD(liq))

	if holders > 0 {
		stats = append(stats, labelStyle.Render("Holders")+FormatNumber(holders))
	}

	if address != "" {
		stats = append(stats, labelStyle.Render("Address")+ui.DimStyle.Render(TruncateAddress(address)))
	}

	if chainID != "" {
		stats = append(stats, labelStyle.Render("Chain")+ui.DimStyle.Render(chainID))
	}

	if launchpad != "" {
		stats = append(stats, labelStyle.Render("Launchpad")+ui.DimStyle.Render(launchpad))
	}

	statsSection := strings.Join(stats, "\n")

	// Price changes section
	var sections []string
	sections = append(sections, header, "", statsSection)

	change5m := getFloat(data, "price_change_5m")
	change1h := getFloat(data, "price_change_1h")
	change4h := getFloat(data, "price_change_4h")
	change24h := getFloat(data, "price_change_24h")

	if change5m != 0 || change1h != 0 || change4h != 0 || change24h != 0 {
		changeTitle := lipgloss.NewStyle().
			Foreground(ui.ColorBright).
			Bold(true).
			Render("Price Changes")

		changeLabelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(10)
		var changeLines []string
		if change5m != 0 {
			changeLines = append(changeLines, "  "+changeLabelStyle.Render("5m")+FormatPercent(change5m))
		}
		if change1h != 0 {
			changeLines = append(changeLines, "  "+changeLabelStyle.Render("1h")+FormatPercent(change1h))
		}
		if change4h != 0 {
			changeLines = append(changeLines, "  "+changeLabelStyle.Render("4h")+FormatPercent(change4h))
		}
		if change24h != 0 {
			changeLines = append(changeLines, "  "+changeLabelStyle.Render("24h")+FormatPercent(change24h))
		}

		if len(changeLines) > 0 {
			sections = append(sections, "", changeTitle)
			sections = append(sections, changeLines...)
		}
	}

	// Security audit section
	if secData, ok := data["security"].(map[string]any); ok {
		secTitle := lipgloss.NewStyle().
			Foreground(ui.ColorGold).
			Bold(true).
			Render("Security Audit")

		var secLines []string

		honeypot, hpOk := getBool(secData, "honeypot")
		if hpOk {
			icon := ui.SuccessStyle.Render("✓")
			label := "Not Honeypot"
			if honeypot {
				icon = ui.ErrorStyle.Render("✗")
				label = "Honeypot"
			}
			secLines = append(secLines, fmt.Sprintf("  %s  %s", icon, label))
		}

		mintable, mOk := getBool(secData, "mintable")
		if mOk {
			icon := ui.SuccessStyle.Render("✓")
			label := "Not Mintable"
			if mintable {
				icon = ui.ErrorStyle.Render("✗")
				label = "Mintable"
			}
			secLines = append(secLines, fmt.Sprintf("  %s  %s", icon, label))
		}

		blacklist, blOk := getBool(secData, "blacklist")
		if blOk {
			icon := ui.SuccessStyle.Render("✓")
			label := "No Blacklist"
			if blacklist {
				icon = ui.ErrorStyle.Render("✗")
				label = "Has Blacklist"
			}
			secLines = append(secLines, fmt.Sprintf("  %s  %s", icon, label))
		}

		buyTax := getFloat(secData, "buy_tax")
		sellTax := getFloat(secData, "sell_tax")
		if buyTax > 0 || sellTax > 0 {
			secLines = append(secLines, fmt.Sprintf("  Buy Tax:  %.1f%%", buyTax))
			secLines = append(secLines, fmt.Sprintf("  Sell Tax: %.1f%%", sellTax))
		}

		if len(secLines) > 0 {
			sections = append(sections, "", secTitle)
			sections = append(sections, secLines...)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return ui.BoxBorder.Render(content)
}

// FormatBrewingTokens renders a table of brewing/newly launched tokens with
// graduation progress bars.
// Response: { "table", "chain", "count", "tokens": [{ "symbol", "price_usd",
//   "market_cap", "liquidity", "graduation_percent", "launchpad", "age_minutes" }] }
func FormatBrewingTokens(data map[string]any) string {
	tokens, _ := data["tokens"].([]any)
	if len(tokens) == 0 {
		return ui.DimStyle.Render("No brewing tokens found.")
	}

	table := getString(data, "table")
	chain := getString(data, "chain")

	maxRows := 15
	if len(tokens) > maxRows {
		tokens = tokens[:maxRows]
	}

	compact := isCompact()

	var colSymbol, colPrice, colMcap, colLiq, colGrad int
	if compact {
		colSymbol = 10
		colPrice = 12
		colMcap = 12
		colGrad = 12
	} else {
		colSymbol = 12
		colPrice = 14
		colMcap = 14
		colLiq = 14
		colGrad = 14
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

	headerParts := []string{
		headerStyle.Width(colSymbol).Render("Symbol"),
		headerStyle.Width(colPrice).Render("Price"),
		headerStyle.Width(colMcap).Render("Mkt Cap"),
	}
	if !compact {
		headerParts = append(headerParts, headerStyle.Width(colLiq).Render("Liquidity"))
	}
	headerParts = append(headerParts, headerStyle.Width(colGrad).Render("Grad %"))
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := colSymbol + colPrice + colMcap + colGrad
	if !compact {
		totalCols += colLiq
	}

	var rows []string
	rows = append(rows, header)
	rows = append(rows, sepLine(totalCols))

	for _, t := range tokens {
		token, ok := t.(map[string]any)
		if !ok {
			continue
		}

		symbol := getString(token, "symbol")
		price := getFloat(token, "price_usd")
		mcap := getFloat(token, "market_cap")
		liq := getFloat(token, "liquidity")
		// graduation_percent may come from launchpadGraduationPercent or graduationPercent
		gradPct := getFloat(token, "graduation_percent")
		if gradPct == 0 {
			gradPct = getFloat(token, "graduation_progress")
		}
		if gradPct == 0 {
			gradPct = getFloat(token, "grad_percent")
		}
		ageMins := getFloat(token, "age_minutes")
		if ageMins == 0 {
			ageSecs := getFloat(token, "age_seconds")
			if ageSecs > 0 {
				ageMins = ageSecs / 60
			}
		}

		symbolLabel := symbol
		if !compact {
			if ageMins > 0 && ageMins < 60 {
				symbolLabel = fmt.Sprintf("%s %dm", symbol, int(ageMins))
			} else if ageMins >= 60 {
				symbolLabel = fmt.Sprintf("%s %dh", symbol, int(ageMins/60))
			}
		}

		barW := 8
		if compact {
			barW = 5
		}
		gradStr := ProgressBar(gradPct, 100, barW) + fmt.Sprintf(" %.0f%%", gradPct)

		rowParts := []string{
			lipgloss.NewStyle().Width(colSymbol).Foreground(ui.ColorBright).Render(symbolLabel),
			lipgloss.NewStyle().Width(colPrice).Render(FormatUSD(price)),
			lipgloss.NewStyle().Width(colMcap).Render(FormatUSD(mcap)),
		}
		if !compact {
			rowParts = append(rowParts, lipgloss.NewStyle().Width(colLiq).Render(FormatUSD(liq)))
		}
		_ = liq // used in full mode
		rowParts = append(rowParts, lipgloss.NewStyle().Width(colGrad).Render(gradStr))
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	titleText := "BREWING"
	if table != "" {
		titleText += " — " + table
	}
	if chain != "" {
		titleText += " (" + chain + ")"
	}
	title := ui.TitleStyle.Render(titleText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// FormatTokenPrice renders a simple price list for one or more tokens.
// Response: { "prices": [{ "address", "price_usd", "price_change_24h", "volume_24h", "market_cap" }] }
func FormatTokenPrice(data map[string]any) string {
	prices, _ := data["prices"].([]any)
	if len(prices) == 0 {
		return ui.DimStyle.Render("No price data available.")
	}

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(14)
	var sections []string

	for _, p := range prices {
		item, ok := p.(map[string]any)
		if !ok {
			continue
		}

		address := getString(item, "address")
		price := getFloat(item, "price_usd")
		change := getFloat(item, "price_change_24h")
		vol := getFloat(item, "volume_24h")
		mcap := getFloat(item, "market_cap")

		var lines []string
		if address != "" {
			lines = append(lines, labelStyle.Render("Address")+ui.DimStyle.Render(TruncateAddress(address)))
		}
		lines = append(lines, labelStyle.Render("Price")+smartFormatPrice(price))
		lines = append(lines, labelStyle.Render("24h Change")+FormatPercent(change))
		if vol > 0 {
			lines = append(lines, labelStyle.Render("Volume 24h")+FormatUSD(vol))
		}
		if mcap > 0 {
			lines = append(lines, labelStyle.Render("Market Cap")+FormatUSD(mcap))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	title := ui.TitleStyle.Render("TOKEN PRICE")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(sections, "\n\n"),
	)

	return ui.BoxBorder.Render(content)
}
