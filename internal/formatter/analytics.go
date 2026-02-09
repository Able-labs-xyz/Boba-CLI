package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatNetworkStats renders network analytics including volume breakdown,
// transaction counts, and liquidity in a bordered box.
// Response: { "network_id", "volume": { "volume_24h", "volume_12h", "volume_4h",
//   "volume_1h", "change_24h" }, "transactions": { "txns_24h", "txns_12h",
//   "txns_1h" }, "liquidity": { "total" }, "summary": "..." }
func FormatNetworkStats(data map[string]any) string {
	header := lipgloss.NewStyle().
		Foreground(ui.ColorGold).
		Bold(true).
		Render("NETWORK STATS")

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(16)
	var sections []string
	sections = append(sections, header, "")

	// Volume breakdown
	volume, _ := data["volume"].(map[string]any)
	if volume != nil {
		volTitle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Volume")
		sections = append(sections, volTitle)

		vol24h := getFloat(volume, "volume_24h")
		vol12h := getFloat(volume, "volume_12h")
		vol4h := getFloat(volume, "volume_4h")
		vol1h := getFloat(volume, "volume_1h")
		change24h := getFloat(volume, "change_24h")

		sections = append(sections, "  "+labelStyle.Render("24h")+FormatUSD(vol24h))
		sections = append(sections, "  "+labelStyle.Render("12h")+FormatUSD(vol12h))
		sections = append(sections, "  "+labelStyle.Render("4h")+FormatUSD(vol4h))
		sections = append(sections, "  "+labelStyle.Render("1h")+FormatUSD(vol1h))
		sections = append(sections, "  "+labelStyle.Render("24h Change")+FormatPercent(change24h))
	}

	// Transaction counts
	txns, _ := data["transactions"].(map[string]any)
	if txns != nil {
		sections = append(sections, "")
		txTitle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Transactions")
		sections = append(sections, txTitle)

		txns24h := getFloat(txns, "txns_24h")
		txns12h := getFloat(txns, "txns_12h")
		txns1h := getFloat(txns, "txns_1h")

		sections = append(sections, "  "+labelStyle.Render("24h")+FormatNumber(txns24h))
		sections = append(sections, "  "+labelStyle.Render("12h")+FormatNumber(txns12h))
		sections = append(sections, "  "+labelStyle.Render("1h")+FormatNumber(txns1h))
	}

	// Liquidity
	liquidity, _ := data["liquidity"].(map[string]any)
	if liquidity != nil {
		total := getFloat(liquidity, "total")
		sections = append(sections, "")
		sections = append(sections, labelStyle.Render("Liquidity")+FormatUSD(total))
	}

	// Summary
	summary := getString(data, "summary")
	if summary != "" {
		sections = append(sections, "")
		sections = append(sections, ui.DimStyle.Render(summary))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return ui.BoxBorder.Render(content)
}

// FormatSearchWallets renders a table of smart wallets with profit, win rate,
// volume, and swap counts.
// Response: { "count", "period", "wallets": [{ "address", "labels",
//   "realized_profit_usd", "win_rate", "volume_usd", "swaps",
//   "bot_score", "scammer_score" }] }
func FormatSearchWallets(data map[string]any) string {
	wallets, _ := data["wallets"].([]any)
	if len(wallets) == 0 {
		return ui.DimStyle.Render("No wallets found.")
	}

	period := getString(data, "period")

	maxRows := 10
	if len(wallets) > maxRows {
		wallets = wallets[:maxRows]
	}

	compact := isCompact()

	var colAddr, colProfit, colWinRate, colVol, colSwaps int
	if compact {
		colAddr = 12
		colProfit = 12
		colWinRate = 10
		colSwaps = 8
	} else {
		colAddr = 14
		colProfit = 14
		colWinRate = 12
		colVol = 14
		colSwaps = 10
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

	headerParts := []string{
		headerStyle.Width(colAddr).Render("Address"),
		headerStyle.Width(colProfit).Render("Profit"),
		headerStyle.Width(colWinRate).Render("Win Rate"),
	}
	if !compact {
		headerParts = append(headerParts, headerStyle.Width(colVol).Render("Volume"))
	}
	headerParts = append(headerParts, headerStyle.Width(colSwaps).Render("Swaps"))
	tableHeader := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := colAddr + colProfit + colWinRate + colSwaps
	if !compact {
		totalCols += colVol
	}

	var rows []string
	rows = append(rows, tableHeader)
	rows = append(rows, sepLine(totalCols))

	for _, w := range wallets {
		wallet, ok := w.(map[string]any)
		if !ok {
			continue
		}

		address := getString(wallet, "address")
		profit := getFloat(wallet, "realized_profit_usd")
		vol := getFloat(wallet, "volume_usd")
		swaps := getFloat(wallet, "swaps")

		// win_rate comes as string like "68.5" or "68.5%" from API
		winRate := getFloat(wallet, "win_rate")

		profitStyle := lipgloss.NewStyle().Width(colProfit)
		if profit >= 0 {
			profitStyle = profitStyle.Foreground(ui.ColorGreen)
		} else {
			profitStyle = profitStyle.Foreground(ui.ColorRed)
		}

		rowParts := []string{
			lipgloss.NewStyle().Width(colAddr).Foreground(ui.ColorBright).Render(TruncateAddress(address)),
			profitStyle.Render(FormatUSD(profit)),
			lipgloss.NewStyle().Width(colWinRate).Render(fmt.Sprintf("%.1f%%", winRate)),
		}
		if !compact {
			rowParts = append(rowParts, lipgloss.NewStyle().Width(colVol).Render(FormatUSD(vol)))
		}
		_ = vol // used in full mode
		rowParts = append(rowParts, lipgloss.NewStyle().Width(colSwaps).Render(FormatNumber(swaps)))
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	titleText := "SMART WALLETS"
	if period != "" {
		titleText += " (" + period + ")"
	}
	title := ui.TitleStyle.Render(titleText)

	var sections []string
	sections = append(sections, title, "", strings.Join(rows, "\n"))

	// Hint from API
	hint := getString(data, "hint")
	if hint != "" {
		sections = append(sections, "", ui.DimStyle.Render(hint))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return ui.BoxBorder.Render(content)
}

// FormatWalletStats renders detailed stats for a single wallet across multiple
// time periods (1d, 1w, 30d) with labels and an optional insight.
// Response: { "wallet_address", "labels": [...], "bot_score",
//   "stats_1d": { "realized_profit_usd", "win_rate", "volume_usd", "swaps" },
//   "stats_1w": {...}, "stats_30d": {...}, "insight": "..." }
func FormatWalletStats(data map[string]any) string {
	walletAddr := getString(data, "wallet_address")
	botScore := getFloat(data, "bot_score")

	header := lipgloss.NewStyle().
		Foreground(ui.ColorBoba).
		Bold(true).
		Render("WALLET STATS")

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(14)
	var sections []string
	sections = append(sections, header, "")

	// Wallet address
	sections = append(sections, labelStyle.Render("Address")+ui.DimStyle.Render(TruncateAddress(walletAddr)))

	// Labels
	labels, _ := data["labels"].([]any)
	if len(labels) > 0 {
		var tags []string
		tagStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorBoba).
			Padding(0, 1)
		for _, l := range labels {
			if s, ok := l.(string); ok {
				tags = append(tags, tagStyle.Render(s))
			}
		}
		if len(tags) > 0 {
			sections = append(sections, labelStyle.Render("Labels")+strings.Join(tags, " "))
		}
	}

	if botScore > 0 {
		sections = append(sections, labelStyle.Render("Bot Score")+fmt.Sprintf("%.0f/10", botScore))
	}

	// Stats table across periods
	periods := []struct {
		key   string
		label string
	}{
		{"stats_1d", "1d"},
		{"stats_1w", "1w"},
		{"stats_30d", "30d"},
	}

	// Check if any period data exists
	var hasPeriodData bool
	for _, p := range periods {
		if _, ok := data[p.key].(map[string]any); ok {
			hasPeriodData = true
			break
		}
	}

	if hasPeriodData {
		sections = append(sections, "")

		colLabel := 12
		colPeriod := 12
		if !isCompact() {
			colLabel = 14
			colPeriod = 14
		}
		headerSt := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

		// Period header row
		periodHeader := lipgloss.NewStyle().Width(colLabel).Render("")
		for _, p := range periods {
			periodHeader += headerSt.Width(colPeriod).Render(p.label)
		}
		sections = append(sections, periodHeader)
		sections = append(sections, sepLine(colLabel+colPeriod*len(periods)))

		// Stat rows
		statFields := []struct {
			key   string
			label string
		}{
			{"realized_profit_usd", "Profit"},
			{"win_rate", "Win Rate"},
			{"volume_usd", "Volume"},
			{"swaps", "Swaps"},
		}

		for _, sf := range statFields {
			row := lipgloss.NewStyle().Width(colLabel).Foreground(ui.ColorBright).Bold(true).Render(sf.label)
			for _, p := range periods {
				periodData, _ := data[p.key].(map[string]any)
				val := getFloat(periodData, sf.key)
				var cell string
				switch sf.key {
				case "realized_profit_usd":
					cell = FormatUSD(val)
				case "win_rate":
					cell = fmt.Sprintf("%.1f%%", val)
				case "volume_usd":
					cell = FormatUSD(val)
				case "swaps":
					cell = FormatNumber(val)
				}
				row += lipgloss.NewStyle().Width(colPeriod).Render(cell)
			}
			sections = append(sections, row)
		}
	}

	// Insight
	insight := getString(data, "insight")
	if insight != "" {
		sections = append(sections, "")
		sections = append(sections, ui.DimStyle.Render(insight))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return ui.BoxBorder.Render(content)
}

// FormatHolders renders a table of top token holders with buy/sell activity
// and realized profit.
// Response: { "token", "chain_id", "holder_count", "summary": { "total_bought_usd",
//   "total_sold_usd" }, "holders": [{ "address", "bought_usd", "sold_usd",
//   "buy_count", "sell_count", "realized_profit_usd", "realized_profit_pct" }] }
func FormatHolders(data map[string]any) string {
	holders, _ := data["holders"].([]any)
	if len(holders) == 0 {
		return ui.DimStyle.Render("No holder data available.")
	}

	token := getString(data, "token")

	maxRows := 15
	if len(holders) > maxRows {
		holders = holders[:maxRows]
	}

	compact := isCompact()

	var colAddr, colBought, colSold, colBuys, colSells, colProfit, colProfPct int
	if compact {
		// Compact: drop Buys/Sells columns
		colAddr = 12
		colBought = 12
		colSold = 12
		colProfit = 12
		colProfPct = 10
	} else {
		colAddr = 14
		colBought = 14
		colSold = 14
		colBuys = 8
		colSells = 8
		colProfit = 14
		colProfPct = 10
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

	headerParts := []string{
		headerStyle.Width(colAddr).Render("Address"),
		headerStyle.Width(colBought).Render("Bought $"),
		headerStyle.Width(colSold).Render("Sold $"),
	}
	if !compact {
		headerParts = append(headerParts,
			headerStyle.Width(colBuys).Render("Buys"),
			headerStyle.Width(colSells).Render("Sells"),
		)
	}
	headerParts = append(headerParts,
		headerStyle.Width(colProfit).Render("Profit $"),
		headerStyle.Width(colProfPct).Render("PnL %"),
	)
	tableHeader := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := colAddr + colBought + colSold + colProfit + colProfPct
	if !compact {
		totalCols += colBuys + colSells
	}

	var rows []string
	rows = append(rows, tableHeader)
	rows = append(rows, sepLine(totalCols))

	for _, h := range holders {
		holder, ok := h.(map[string]any)
		if !ok {
			continue
		}

		address := getString(holder, "address")
		bought := getFloat(holder, "bought_usd")
		sold := getFloat(holder, "sold_usd")
		buys := getFloat(holder, "buy_count")
		sells := getFloat(holder, "sell_count")
		profit := getFloat(holder, "realized_profit_usd")
		profitPct := getFloat(holder, "realized_profit_pct")

		profitStyle := lipgloss.NewStyle().Width(colProfit)
		if profit >= 0 {
			profitStyle = profitStyle.Foreground(ui.ColorGreen)
		} else {
			profitStyle = profitStyle.Foreground(ui.ColorRed)
		}

		rowParts := []string{
			lipgloss.NewStyle().Width(colAddr).Foreground(ui.ColorBright).Render(TruncateAddress(address)),
			lipgloss.NewStyle().Width(colBought).Render(FormatUSD(bought)),
			lipgloss.NewStyle().Width(colSold).Render(FormatUSD(sold)),
		}
		if !compact {
			rowParts = append(rowParts,
				lipgloss.NewStyle().Width(colBuys).Render(FormatNumber(buys)),
				lipgloss.NewStyle().Width(colSells).Render(FormatNumber(sells)),
			)
		}
		_ = buys  // used in full mode
		_ = sells // used in full mode
		rowParts = append(rowParts,
			profitStyle.Render(FormatUSD(profit)),
			lipgloss.NewStyle().Width(colProfPct).Render(FormatPercent(profitPct)),
		)
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	// Summary line
	summary, _ := data["summary"].(map[string]any)
	if summary != nil {
		totalBought := getFloat(summary, "total_bought_usd")
		totalSold := getFloat(summary, "total_sold_usd")
		rows = append(rows, sepLine(totalCols))
		rows = append(rows, fmt.Sprintf("Total Bought: %s  |  Total Sold: %s", FormatUSD(totalBought), FormatUSD(totalSold)))
	}

	titleText := "TOP HOLDERS"
	if token != "" {
		titleText += " — " + TruncateAddress(token)
	}
	title := ui.TitleStyle.Render(titleText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// FormatDeployerTokens renders a table of tokens deployed by a specific address.
// Response: { "deployer", "count", "tokens": [{ "address", "name", "symbol",
//   "price_usd", "market_cap", "created_at" }] }
func FormatDeployerTokens(data map[string]any) string {
	tokens, _ := data["tokens"].([]any)
	if len(tokens) == 0 {
		return ui.DimStyle.Render("No deployer tokens found.")
	}

	deployer := getString(data, "deployer")

	compact := isCompact()

	var colSymbol, colPrice, colMcap, colAddr int
	if compact {
		colSymbol = 10
		colPrice = 12
		colMcap = 12
	} else {
		colSymbol = 12
		colPrice = 14
		colMcap = 14
		colAddr = 14
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

	headerParts := []string{
		headerStyle.Width(colSymbol).Render("Symbol"),
		headerStyle.Width(colPrice).Render("Price"),
		headerStyle.Width(colMcap).Render("Mkt Cap"),
	}
	if !compact {
		headerParts = append(headerParts, headerStyle.Width(colAddr).Render("Address"))
	}
	tableHeader := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := colSymbol + colPrice + colMcap
	if !compact {
		totalCols += colAddr
	}

	var rows []string
	rows = append(rows, tableHeader)
	rows = append(rows, sepLine(totalCols))

	for _, t := range tokens {
		token, ok := t.(map[string]any)
		if !ok {
			continue
		}

		symbol := getString(token, "symbol")
		price := getFloat(token, "price_usd")
		mcap := getFloat(token, "market_cap")
		address := getString(token, "address")

		rowParts := []string{
			lipgloss.NewStyle().Width(colSymbol).Foreground(ui.ColorBright).Render(symbol),
			lipgloss.NewStyle().Width(colPrice).Render(FormatUSD(price)),
			lipgloss.NewStyle().Width(colMcap).Render(FormatUSD(mcap)),
		}
		if !compact {
			rowParts = append(rowParts, lipgloss.NewStyle().Width(colAddr).Render(ui.DimStyle.Render(TruncateAddress(address))))
		}
		_ = address // used in full mode
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	titleText := "DEPLOYER TOKENS"
	if deployer != "" {
		titleText += " — " + TruncateAddress(deployer)
	}
	title := ui.TitleStyle.Render(titleText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// FormatDeployerActivity renders a list of deployer (dev) activity on a token,
// showing buys and sells with amounts and transaction hashes.
// Response: { "deployer", "token", "activity": [{ "type", "timestamp",
//   "amount_usd", "tx_hash" }] }
func FormatDeployerActivity(data map[string]any) string {
	activities, _ := data["activity"].([]any)
	if len(activities) == 0 {
		return ui.DimStyle.Render("No deployer activity found.")
	}

	deployer := getString(data, "deployer")
	token := getString(data, "token")

	var lines []string
	for _, a := range activities {
		activity, ok := a.(map[string]any)
		if !ok {
			continue
		}

		actType := getString(activity, "type")
		amount := getFloat(activity, "amount_usd")
		txHash := getString(activity, "tx_hash")
		timestamp := getString(activity, "timestamp")

		var icon string
		var amountStyle lipgloss.Style
		switch strings.ToLower(actType) {
		case "buy":
			icon = ui.SuccessStyle.Render("▲")
			amountStyle = lipgloss.NewStyle().Foreground(ui.ColorGreen)
		case "sell":
			icon = ui.ErrorStyle.Render("▼")
			amountStyle = lipgloss.NewStyle().Foreground(ui.ColorRed)
		default:
			icon = ui.DimStyle.Render("●")
			amountStyle = lipgloss.NewStyle().Foreground(ui.ColorBright)
		}

		line := fmt.Sprintf("%s  %s  %s",
			icon,
			amountStyle.Render(FormatUSD(amount)),
			ui.DimStyle.Render(TruncateAddress(txHash)),
		)
		if timestamp != "" {
			line += "  " + ui.DimStyle.Render(timestamp)
		}
		lines = append(lines, line)
	}

	titleText := "DEV ACTIVITY"
	if deployer != "" {
		titleText += " — " + TruncateAddress(deployer)
	}
	if token != "" {
		titleText += " [" + TruncateAddress(token) + "]"
	}
	title := ui.TitleStyle.Render(titleText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(lines, "\n"),
	)

	return ui.BoxBorder.Render(content)
}
