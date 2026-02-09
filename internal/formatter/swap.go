package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatSwapQuote renders a swap quote showing the from/to amounts, price
// impact, and estimated gas cost.
func FormatSwapQuote(data map[string]any) string {
	fromAmount := getFloat(data, "from_amount")
	fromSymbol := getString(data, "from_symbol")
	toAmount := getFloat(data, "to_amount")
	toSymbol := getString(data, "to_symbol")
	priceImpact := getFloat(data, "price_impact")
	gasEstimate := getFloat(data, "gas_estimate")

	title := ui.TitleStyle.Render("SWAP QUOTE")

	amountStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	symbolStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba)

	fromLine := fmt.Sprintf("FROM:  %s %s",
		amountStyle.Render(FormatNumber(fromAmount)),
		symbolStyle.Render(fromSymbol),
	)

	arrow := lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true).Render("  →")

	toLine := fmt.Sprintf("TO:    %s %s",
		amountStyle.Render(FormatNumber(toAmount)),
		symbolStyle.Render(toSymbol),
	)

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(16)

	impactLine := labelStyle.Render("Price Impact") + FormatPercent(priceImpact)
	gasLine := labelStyle.Render("Est. Gas") + FormatUSD(gasEstimate)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		fromLine,
		arrow,
		toLine,
		"",
		impactLine,
		gasLine,
	)

	return ui.BoxBorder.Render(content)
}

// FormatTradeResult renders the result of a swap execution. It shows a success
// box with transaction details or a failure box with the error message.
func FormatTradeResult(data map[string]any) string {
	// Check for error
	if errMsg := getString(data, "error"); errMsg != "" {
		return formatTradeFailed(errMsg)
	}

	// Check for explicit success field
	if success, ok := getBool(data, "success"); ok && !success {
		errMsg := getString(data, "error")
		if errMsg == "" {
			errMsg = getString(data, "message")
		}
		if errMsg == "" {
			errMsg = "Unknown error"
		}
		return formatTradeFailed(errMsg)
	}

	return formatTradeSuccess(data)
}

// formatTradeSuccess renders a successful trade with green styling.
func formatTradeSuccess(data map[string]any) string {
	header := lipgloss.NewStyle().
		Foreground(ui.ColorGreen).
		Bold(true).
		Render("TRADE EXECUTED ✓")

	var lines []string
	lines = append(lines, header, "")

	txHash := getString(data, "tx_hash")
	if txHash == "" {
		txHash = getString(data, "hash")
	}
	if txHash == "" {
		txHash = getString(data, "transaction_hash")
	}
	if txHash != "" {
		labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(14)
		lines = append(lines, labelStyle.Render("Tx Hash")+ui.DimStyle.Render(TruncateAddress(txHash)))
	}

	fromAmount := getFloat(data, "from_amount")
	fromSymbol := getString(data, "from_symbol")
	toAmount := getFloat(data, "to_amount")
	toSymbol := getString(data, "to_symbol")

	if fromSymbol != "" && toSymbol != "" {
		swapLine := fmt.Sprintf("Swapped %s %s → %s %s",
			FormatNumber(fromAmount), fromSymbol,
			FormatNumber(toAmount), toSymbol,
		)
		lines = append(lines, swapLine)
	}

	fromAddr := getString(data, "from_address")
	toAddr := getString(data, "to_address")
	if fromAddr == "" {
		fromAddr = getString(data, "from_token")
	}
	if toAddr == "" {
		toAddr = getString(data, "to_token")
	}

	if fromAddr != "" {
		labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(14)
		lines = append(lines, labelStyle.Render("From Token")+ui.DimStyle.Render(TruncateAddress(fromAddr)))
	}
	if toAddr != "" {
		labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(14)
		lines = append(lines, labelStyle.Render("To Token")+ui.DimStyle.Render(TruncateAddress(toAddr)))
	}

	content := strings.Join(lines, "\n")
	return ui.SuccessBoxBorder.Render(content)
}

// formatTradeFailed renders a failed trade with red styling and error message.
func formatTradeFailed(errMsg string) string {
	header := lipgloss.NewStyle().
		Foreground(ui.ColorRed).
		Bold(true).
		Render("TRADE FAILED ✗")

	errorLine := ui.ErrorStyle.Render(errMsg)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		errorLine,
	)

	return ui.ErrorBoxBorder.Render(content)
}
