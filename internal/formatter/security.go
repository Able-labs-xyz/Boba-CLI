package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatAuditToken renders a full security audit for a single token.
// Handles both Solana and EVM response formats with security checks,
// holder analysis, tax info, and liquidity data.
func FormatAuditToken(data map[string]any) string {
	token := getString(data, "token")
	riskLevel := getString(data, "risk_level")

	// Title
	title := lipgloss.NewStyle().
		Foreground(ui.ColorBoba).
		Bold(true).
		Render("SECURITY AUDIT")

	// Token address
	var addressLine string
	if token != "" {
		addressLine = lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(14).Render("Token") +
			ui.DimStyle.Render(TruncateAddress(token))
	}

	// Risk level badge
	riskBadge := renderRiskBadge(riskLevel)

	// Security checks
	secData, _ := data["security"].(map[string]any)
	var secLines []string
	if secData != nil {
		secLines = append(secLines, lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Security Checks"))

		// Common checks
		if hp, ok := getBool(secData, "is_honeypot"); ok {
			secLines = append(secLines, boolCheck(!hp, "Not Honeypot", "Honeypot"))
		}
		if mint, ok := getBool(secData, "is_mintable"); ok {
			secLines = append(secLines, boolCheck(!mint, "Not Mintable", "Mintable"))
		}

		// Solana-specific
		if freeze, ok := getBool(secData, "freezable"); ok {
			secLines = append(secLines, boolCheck(!freeze, "Not Freezable", "Freezable"))
		}
		if grad, ok := getBool(secData, "graduated"); ok {
			secLines = append(secLines, boolCheck(grad, "Graduated", "Not Graduated"))
		}

		// EVM-specific
		if proxy, ok := getBool(secData, "is_proxy"); ok {
			secLines = append(secLines, boolCheck(!proxy, "Not Proxy", "Proxy Contract"))
		}
		if takeback, ok := getBool(secData, "can_take_back_ownership"); ok {
			secLines = append(secLines, boolCheck(!takeback, "No Ownership Takeback", "Can Take Back Ownership"))
		}
		if hidden, ok := getBool(secData, "hidden_owner"); ok {
			secLines = append(secLines, boolCheck(!hidden, "No Hidden Owner", "Hidden Owner"))
		}
	}

	// Holder analysis
	holderData, _ := data["holder_analysis"].(map[string]any)
	var holderLines []string
	if holderData != nil {
		holderLines = append(holderLines, lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Holder Analysis"))

		labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(20)

		top10 := getFloat(holderData, "top10_holders_percent")
		holderLines = append(holderLines, "  "+labelStyle.Render("Top 10 Holders")+fmt.Sprintf("%.1f%%", top10))

		devHold := getFloat(holderData, "dev_holding_percent")
		holderLines = append(holderLines, "  "+labelStyle.Render("Dev Holding")+fmt.Sprintf("%.1f%%", devHold))

		sniperHeld := getFloat(holderData, "sniper_held_percent")
		holderLines = append(holderLines, "  "+labelStyle.Render("Sniper Held")+fmt.Sprintf("%.1f%%", sniperHeld))

		bundlerHeld := getFloat(holderData, "bundler_held_percent")
		holderLines = append(holderLines, "  "+labelStyle.Render("Bundler Held")+fmt.Sprintf("%.1f%%", bundlerHeld))

		holderCount := getFloat(holderData, "holder_count")
		holderLines = append(holderLines, "  "+labelStyle.Render("Holder Count")+FormatNumber(holderCount))
	}

	// Taxes (EVM)
	taxData, _ := data["taxes"].(map[string]any)
	var taxLines []string
	if taxData != nil {
		labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(20)
		taxLines = append(taxLines, lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Taxes"))

		buyTax := getFloat(taxData, "buy_tax")
		sellTax := getFloat(taxData, "sell_tax")
		transferTax := getFloat(taxData, "transfer_tax")
		taxLines = append(taxLines, "  "+labelStyle.Render("Buy Tax")+fmt.Sprintf("%.1f%%", buyTax))
		taxLines = append(taxLines, "  "+labelStyle.Render("Sell Tax")+fmt.Sprintf("%.1f%%", sellTax))
		if transferTax > 0 {
			taxLines = append(taxLines, "  "+labelStyle.Render("Transfer Tax")+fmt.Sprintf("%.1f%%", transferTax))
		}
	}

	// Liquidity
	liqData, _ := data["liquidity"].(map[string]any)
	var liqLines []string
	if liqData != nil {
		liqLines = append(liqLines, lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Render("Liquidity"))

		if locked, ok := getBool(liqData, "lp_locked"); ok {
			liqLines = append(liqLines, boolCheck(locked, "LP Locked", "LP Not Locked"))
		}
		lockDuration := getString(liqData, "lp_lock_duration")
		if lockDuration != "" {
			labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Width(20)
			liqLines = append(liqLines, "  "+labelStyle.Render("Lock Duration")+lockDuration)
		}
	}

	// Assemble sections
	var sections []string
	sections = append(sections, title)
	if addressLine != "" {
		sections = append(sections, addressLine)
	}
	sections = append(sections, riskBadge)

	if len(secLines) > 0 {
		sections = append(sections, "", strings.Join(secLines, "\n"))
	}
	if len(holderLines) > 0 {
		sections = append(sections, "", strings.Join(holderLines, "\n"))
	}
	if len(taxLines) > 0 {
		sections = append(sections, "", strings.Join(taxLines, "\n"))
	}
	if len(liqLines) > 0 {
		sections = append(sections, "", strings.Join(liqLines, "\n"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Pick border based on risk level
	switch strings.ToUpper(riskLevel) {
	case "HIGH":
		return ui.ErrorBoxBorder.Render(content)
	case "MEDIUM":
		return ui.BoxBorder.Render(content)
	default:
		return ui.BoxBorder.Render(content)
	}
}

// FormatAuditBatch renders a batch of token audits as a table with risk-colored rows.
// Response: { "chain", "count", "audits": [{ "token", "is_honeypot", "is_mintable",
// "top10_holders_percent", "lp_locked", "risk_level" }] }
func FormatAuditBatch(data map[string]any) string {
	audits, _ := data["audits"].([]any)
	if len(audits) == 0 {
		return ui.DimStyle.Render("No audit data available.")
	}

	compact := isCompact()

	var colToken, colHP, colMint, colTop10, colLP, colRisk int
	if compact {
		colToken = 14
		colHP = 10
		colMint = 10
		colRisk = 10
	} else {
		colToken = 16
		colHP = 12
		colMint = 12
		colTop10 = 10
		colLP = 10
		colRisk = 10
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorBright)

	headerParts := []string{
		headerStyle.Width(colToken).Render("Token"),
		headerStyle.Width(colHP).Render("Honeypot"),
		headerStyle.Width(colMint).Render("Mintable"),
	}
	if !compact {
		headerParts = append(headerParts,
			headerStyle.Width(colTop10).Render("Top10%"),
			headerStyle.Width(colLP).Render("LP Lock"),
		)
	}
	headerParts = append(headerParts, headerStyle.Width(colRisk).Render("Risk"))
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := colToken + colHP + colMint + colRisk
	if !compact {
		totalCols += colTop10 + colLP
	}

	var rows []string
	rows = append(rows, header)
	rows = append(rows, sepLine(totalCols))

	for _, a := range audits {
		audit, ok := a.(map[string]any)
		if !ok {
			continue
		}

		token := TruncateAddress(getString(audit, "token"))
		risk := getString(audit, "risk_level")

		hp, hpOk := getBool(audit, "is_honeypot")
		hpStr := ui.DimStyle.Render("—")
		if hpOk {
			if hp {
				hpStr = ui.ErrorStyle.Render("YES")
			} else {
				hpStr = ui.SuccessStyle.Render("NO")
			}
		}

		mint, mintOk := getBool(audit, "is_mintable")
		mintStr := ui.DimStyle.Render("—")
		if mintOk {
			if mint {
				mintStr = ui.ErrorStyle.Render("YES")
			} else {
				mintStr = ui.SuccessStyle.Render("NO")
			}
		}

		top10 := getFloat(audit, "top10_holders_percent")
		top10Str := fmt.Sprintf("%.1f%%", top10)

		lpLocked, lpOk := getBool(audit, "lp_locked")
		lpStr := ui.DimStyle.Render("—")
		if lpOk {
			if lpLocked {
				lpStr = ui.SuccessStyle.Render("YES")
			} else {
				lpStr = ui.ErrorStyle.Render("NO")
			}
		}

		riskStyled := renderRiskText(risk)

		rowParts := []string{
			lipgloss.NewStyle().Width(colToken).Foreground(ui.ColorBright).Render(token),
			lipgloss.NewStyle().Width(colHP).Render(hpStr),
			lipgloss.NewStyle().Width(colMint).Render(mintStr),
		}
		if !compact {
			rowParts = append(rowParts,
				lipgloss.NewStyle().Width(colTop10).Render(top10Str),
				lipgloss.NewStyle().Width(colLP).Render(lpStr),
			)
		}
		_ = top10Str // used in full mode
		_ = lpStr    // used in full mode
		rowParts = append(rowParts, lipgloss.NewStyle().Width(colRisk).Render(riskStyled))
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	titleText := "BATCH AUDIT"
	chain := getString(data, "chain")
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

// FormatTokenVerified renders a token verification result as a simple status box.
// Response: { "verified": true, "name", "symbol", "source", "tags": [...] }
// or { "verified": false }
func FormatTokenVerified(data map[string]any) string {
	verified, _ := getBool(data, "verified")

	if !verified {
		status := ui.ErrorStyle.Render("NOT VERIFIED \u2717")
		return ui.ErrorBoxBorder.Render(status)
	}

	name := getString(data, "name")
	symbol := getString(data, "symbol")
	source := getString(data, "source")

	status := ui.SuccessStyle.Render("VERIFIED \u2713")

	var lines []string
	lines = append(lines, status)

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(10)
	if name != "" {
		lines = append(lines, labelStyle.Render("Name")+name)
	}
	if symbol != "" {
		lines = append(lines, labelStyle.Render("Symbol")+symbol)
	}
	if source != "" {
		lines = append(lines, labelStyle.Render("Source")+ui.DimStyle.Render(source))
	}

	// Tags
	if tagsRaw, ok := data["tags"].([]any); ok && len(tagsRaw) > 0 {
		var tags []string
		for _, t := range tagsRaw {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
		if len(tags) > 0 {
			lines = append(lines, labelStyle.Render("Tags")+ui.DimStyle.Render(strings.Join(tags, ", ")))
		}
	}

	content := strings.Join(lines, "\n")
	return ui.SuccessBoxBorder.Render(content)
}

// boolCheck renders a checkmark or cross line for a boolean security check.
func boolCheck(pass bool, passLabel, failLabel string) string {
	if pass {
		return fmt.Sprintf("  %s  %s", ui.SuccessStyle.Render("\u2713"), passLabel)
	}
	return fmt.Sprintf("  %s  %s", ui.ErrorStyle.Render("\u2717"), failLabel)
}

// renderRiskBadge returns a styled risk level badge.
func renderRiskBadge(level string) string {
	upper := strings.ToUpper(level)
	switch upper {
	case "LOW":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorGreen).
			Bold(true).
			Padding(0, 1).
			Render("RISK: LOW")
	case "MEDIUM":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorGold).
			Bold(true).
			Padding(0, 1).
			Render("RISK: MEDIUM")
	case "HIGH":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorRed).
			Bold(true).
			Padding(0, 1).
			Render("RISK: HIGH")
	default:
		return ui.DimStyle.Render("RISK: " + upper)
	}
}

// renderRiskText returns risk level text with appropriate color.
func renderRiskText(level string) string {
	upper := strings.ToUpper(level)
	switch upper {
	case "LOW":
		return ui.SuccessStyle.Render("LOW")
	case "MEDIUM":
		return lipgloss.NewStyle().Foreground(ui.ColorGold).Render("MEDIUM")
	case "HIGH":
		return ui.ErrorStyle.Render("HIGH")
	default:
		return ui.DimStyle.Render(upper)
	}
}
