package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatOrderCreated renders a success box for create_limit_order,
// create_dca_order, and create_twap_order responses.
func FormatOrderCreated(data map[string]any) string {
	success, hasSuccess := getBool(data, "success")
	if hasSuccess && !success {
		errMsg := getString(data, "message")
		if errMsg == "" {
			errMsg = getString(data, "error")
		}
		if errMsg == "" {
			errMsg = "Order creation failed"
		}
		return formatOrderFailed(errMsg)
	}

	header := lipgloss.NewStyle().
		Foreground(ui.ColorGreen).
		Bold(true).
		Render("ORDER CREATED \u2713")

	var lines []string
	lines = append(lines, header, "")

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(16)

	orderID := getString(data, "order_id")
	if orderID != "" {
		lines = append(lines, labelStyle.Render("Order ID")+ui.DimStyle.Render(orderID))
	}

	status := getString(data, "status")
	if status != "" {
		lines = append(lines, labelStyle.Render("Status")+colorStatus(status))
	}

	chain := getString(data, "chain")
	if chain != "" {
		lines = append(lines, labelStyle.Render("Chain")+chain)
	}

	// Limit order fields
	side := getString(data, "side")
	if side != "" {
		lines = append(lines, labelStyle.Render("Side")+formatSide(side))
	}

	inputToken := getString(data, "input_token")
	outputToken := getString(data, "output_token")
	if inputToken != "" {
		lines = append(lines, labelStyle.Render("Input Token")+ui.DimStyle.Render(TruncateAddress(inputToken)))
	}
	if outputToken != "" {
		lines = append(lines, labelStyle.Render("Output Token")+ui.DimStyle.Render(TruncateAddress(outputToken)))
	}

	inputAmount := getFloat(data, "input_amount")
	if inputAmount > 0 {
		lines = append(lines, labelStyle.Render("Input Amount")+FormatNumber(inputAmount))
	}

	triggerPrice := getFloat(data, "trigger_price")
	if triggerPrice > 0 {
		lines = append(lines, labelStyle.Render("Trigger Price")+FormatUSD(triggerPrice))
	}

	expiresAt := getString(data, "expires_at")
	if expiresAt != "" {
		lines = append(lines, labelStyle.Render("Expires")+ui.DimStyle.Render(expiresAt))
	}

	// DCA order fields
	totalAmount := getFloat(data, "total_amount")
	if totalAmount > 0 {
		lines = append(lines, labelStyle.Render("Total Amount")+FormatNumber(totalAmount))
	}

	amountPerInterval := getFloat(data, "amount_per_interval")
	if amountPerInterval > 0 {
		lines = append(lines, labelStyle.Render("Per Interval")+FormatNumber(amountPerInterval))
	}

	totalIntervals := getFloat(data, "total_intervals")
	if totalIntervals > 0 {
		lines = append(lines, labelStyle.Render("Intervals")+fmt.Sprintf("%.0f", totalIntervals))
	}

	intervalSeconds := getFloat(data, "interval_seconds")
	if intervalSeconds > 0 {
		lines = append(lines, labelStyle.Render("Interval")+formatDuration(intervalSeconds))
	}

	nextExecution := getString(data, "next_execution")
	if nextExecution != "" {
		lines = append(lines, labelStyle.Render("Next Exec")+ui.DimStyle.Render(nextExecution))
	}

	// TWAP order fields
	totalSlices := getFloat(data, "total_slices")
	if totalSlices > 0 {
		lines = append(lines, labelStyle.Render("Total Slices")+fmt.Sprintf("%.0f", totalSlices))
	}

	amountPerSlice := getFloat(data, "amount_per_slice")
	if amountPerSlice > 0 {
		lines = append(lines, labelStyle.Render("Per Slice")+FormatNumber(amountPerSlice))
	}

	durationSeconds := getFloat(data, "duration_seconds")
	if durationSeconds > 0 {
		lines = append(lines, labelStyle.Render("Duration")+formatDuration(durationSeconds))
	}

	// Message
	msg := getString(data, "message")
	if msg != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(msg))
	}

	content := strings.Join(lines, "\n")
	return ui.SuccessBoxBorder.Render(content)
}

// FormatOrders renders a table of orders for get_limit_orders,
// get_dca_orders, and get_twap_orders responses.
func FormatOrders(data map[string]any) string {
	orders, _ := data["orders"].([]any)
	if len(orders) == 0 {
		return ui.DimStyle.Render("No orders found.")
	}

	// Detect order type from fields in the first order
	orderType := detectOrderType(orders)

	header := lipgloss.NewStyle().
		Foreground(ui.ColorBoba).
		Bold(true).
		Render(fmt.Sprintf("%s ORDERS", orderType))

	total := getFloat(data, "total")
	if total == 0 {
		total = float64(len(orders))
	}

	subtitle := ui.DimStyle.Render(fmt.Sprintf("Showing %d of %.0f", len(orders), total))

	compact := isCompact()

	var wID, wStatus, wSide, wTrigger, wInput, wCreated int
	if compact {
		wID = 8
		wStatus = 10
		wSide = 5
		wTrigger = 12
		wInput = 10
	} else {
		wID = 10
		wStatus = 12
		wSide = 6
		wTrigger = 16
		wInput = 14
		wCreated = 12
	}

	headerParts := []string{
		lipgloss.NewStyle().Width(wID).Bold(true).Render("ID"),
		lipgloss.NewStyle().Width(wStatus).Bold(true).Render("Status"),
		lipgloss.NewStyle().Width(wSide).Bold(true).Render("Side"),
		lipgloss.NewStyle().Width(wTrigger).Bold(true).Render("Trigger $"),
		lipgloss.NewStyle().Width(wInput).Bold(true).Render("Amount"),
	}
	if !compact {
		headerParts = append(headerParts, lipgloss.NewStyle().Width(wCreated).Bold(true).Render("Created"))
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := wID + wStatus + wSide + wTrigger + wInput
	if !compact {
		totalCols += wCreated
	}

	var rows []string
	rows = append(rows, headerRow)
	rows = append(rows, sepLine(totalCols))

	maxRows := 10
	showMore := len(orders) > maxRows
	displayed := orders
	if len(orders) > maxRows {
		displayed = orders[:maxRows]
	}

	for _, o := range displayed {
		order, ok := o.(map[string]any)
		if !ok {
			continue
		}

		id := getString(order, "id")
		if len(id) > 8 {
			id = id[:8]
		}

		status := getString(order, "status")
		side := getString(order, "side")
		triggerPrice := getFloat(order, "trigger_price")
		inputAmount := getFloat(order, "input_amount")
		createdAt := getString(order, "created_at")
		if len(createdAt) > 10 {
			createdAt = createdAt[:10]
		}

		// Use smartFormatPrice for trigger (handles very small token prices)
		triggerStr := smartFormatPrice(triggerPrice)
		if triggerPrice == 0 {
			triggerStr = ui.DimStyle.Render("—")
		}

		inputStr := FormatNumber(inputAmount)
		if inputAmount == 0 {
			inputStr = ui.DimStyle.Render("—")
		}

		rowParts := []string{
			lipgloss.NewStyle().Width(wID).Foreground(ui.ColorBright).Render(id),
			lipgloss.NewStyle().Width(wStatus).Render(colorStatus(status)),
			lipgloss.NewStyle().Width(wSide).Render(formatSide(side)),
			lipgloss.NewStyle().Width(wTrigger).Render(triggerStr),
			lipgloss.NewStyle().Width(wInput).Render(inputStr),
		}
		if !compact {
			rowParts = append(rowParts, lipgloss.NewStyle().Width(wCreated).Render(ui.DimStyle.Render(createdAt)))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	if showMore {
		remaining := len(orders) - maxRows
		rows = append(rows, ui.DimStyle.Render(fmt.Sprintf("...and %d more orders", remaining)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		subtitle,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// FormatOrderDetail renders a detailed view of a single order.
func FormatOrderDetail(data map[string]any) string {
	header := lipgloss.NewStyle().
		Foreground(ui.ColorBoba).
		Bold(true).
		Render("ORDER DETAIL")

	var lines []string
	lines = append(lines, header, "")

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(16)

	id := getString(data, "id")
	if id != "" {
		lines = append(lines, labelStyle.Render("ID")+ui.DimStyle.Render(id))
	}

	status := getString(data, "status")
	if status != "" {
		lines = append(lines, labelStyle.Render("Status")+colorStatus(status))
	}

	chain := getString(data, "chain")
	if chain != "" {
		lines = append(lines, labelStyle.Render("Chain")+chain)
	}

	side := getString(data, "side")
	if side != "" {
		lines = append(lines, labelStyle.Render("Side")+formatSide(side))
	}

	inputToken := getString(data, "input_token")
	outputToken := getString(data, "output_token")
	if inputToken != "" {
		lines = append(lines, labelStyle.Render("Input Token")+ui.DimStyle.Render(TruncateAddress(inputToken)))
	}
	if outputToken != "" {
		lines = append(lines, labelStyle.Render("Output Token")+ui.DimStyle.Render(TruncateAddress(outputToken)))
	}

	inputAmount := getFloat(data, "input_amount")
	if inputAmount > 0 {
		lines = append(lines, labelStyle.Render("Input Amount")+FormatNumber(inputAmount))
	}

	triggerPrice := getFloat(data, "trigger_price")
	if triggerPrice > 0 {
		lines = append(lines, labelStyle.Render("Trigger Price")+FormatUSD(triggerPrice))
	}

	// DCA / TWAP fields
	totalAmount := getFloat(data, "total_amount")
	if totalAmount > 0 {
		lines = append(lines, labelStyle.Render("Total Amount")+FormatNumber(totalAmount))
	}

	amountPerInterval := getFloat(data, "amount_per_interval")
	if amountPerInterval > 0 {
		lines = append(lines, labelStyle.Render("Per Interval")+FormatNumber(amountPerInterval))
	}

	totalIntervals := getFloat(data, "total_intervals")
	if totalIntervals > 0 {
		lines = append(lines, labelStyle.Render("Intervals")+fmt.Sprintf("%.0f", totalIntervals))
	}

	totalSlices := getFloat(data, "total_slices")
	if totalSlices > 0 {
		lines = append(lines, labelStyle.Render("Total Slices")+fmt.Sprintf("%.0f", totalSlices))
	}

	amountPerSlice := getFloat(data, "amount_per_slice")
	if amountPerSlice > 0 {
		lines = append(lines, labelStyle.Render("Per Slice")+FormatNumber(amountPerSlice))
	}

	intervalSeconds := getFloat(data, "interval_seconds")
	if intervalSeconds > 0 {
		lines = append(lines, labelStyle.Render("Interval")+formatDuration(intervalSeconds))
	}

	durationSeconds := getFloat(data, "duration_seconds")
	if durationSeconds > 0 {
		lines = append(lines, labelStyle.Render("Duration")+formatDuration(durationSeconds))
	}

	nextExecution := getString(data, "next_execution")
	if nextExecution != "" {
		lines = append(lines, labelStyle.Render("Next Exec")+ui.DimStyle.Render(nextExecution))
	}

	expiresAt := getString(data, "expires_at")
	if expiresAt != "" {
		lines = append(lines, labelStyle.Render("Expires")+ui.DimStyle.Render(expiresAt))
	}

	// Position-related fields
	entryPrice := getFloat(data, "entry_price")
	if entryPrice > 0 {
		lines = append(lines, labelStyle.Render("Entry Price")+FormatUSD(entryPrice))
	}

	stopLoss := getFloat(data, "stop_loss")
	if stopLoss > 0 {
		lines = append(lines, labelStyle.Render("Stop Loss")+FormatUSD(stopLoss))
	}

	takeProfit := getFloat(data, "take_profit")
	if takeProfit > 0 {
		lines = append(lines, labelStyle.Render("Take Profit")+FormatUSD(takeProfit))
	}

	createdAt := getString(data, "created_at")
	if createdAt != "" {
		lines = append(lines, labelStyle.Render("Created")+ui.DimStyle.Render(createdAt))
	}

	updatedAt := getString(data, "updated_at")
	if updatedAt != "" {
		lines = append(lines, labelStyle.Render("Updated")+ui.DimStyle.Render(updatedAt))
	}

	content := strings.Join(lines, "\n")
	return ui.BoxBorder.Render(content)
}

// FormatOrderAction renders the result of cancel/pause/resume/update actions.
func FormatOrderAction(data map[string]any) string {
	success, hasSuccess := getBool(data, "success")
	if hasSuccess && !success {
		errMsg := getString(data, "message")
		if errMsg == "" {
			errMsg = getString(data, "error")
		}
		if errMsg == "" {
			errMsg = "Action failed"
		}
		return formatOrderFailed(errMsg)
	}

	header := lipgloss.NewStyle().
		Foreground(ui.ColorGreen).
		Bold(true).
		Render("ORDER ACTION \u2713")

	var lines []string
	lines = append(lines, header, "")

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(14)

	orderID := getString(data, "order_id")
	if orderID != "" {
		lines = append(lines, labelStyle.Render("Order ID")+ui.DimStyle.Render(orderID))
	}

	msg := getString(data, "message")
	if msg != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(msg))
	}

	content := strings.Join(lines, "\n")
	return ui.SuccessBoxBorder.Render(content)
}

// FormatPositions renders a table of positions from get_positions.
func FormatPositions(data map[string]any) string {
	positions, _ := data["positions"].([]any)
	if len(positions) == 0 {
		return ui.DimStyle.Render("No positions found.")
	}

	header := lipgloss.NewStyle().
		Foreground(ui.ColorBoba).
		Bold(true).
		Render("POSITIONS")

	compact := isCompact()

	var pID, pToken, pEntry, pSL, pTP, pStatus int
	if compact {
		pID = 8
		pToken = 12
		pEntry = 10
		pStatus = 8
	} else {
		pID = 10
		pToken = 14
		pEntry = 12
		pSL = 12
		pTP = 12
		pStatus = 10
	}

	headerParts := []string{
		lipgloss.NewStyle().Width(pID).Bold(true).Render("ID"),
		lipgloss.NewStyle().Width(pToken).Bold(true).Render("Token"),
		lipgloss.NewStyle().Width(pEntry).Bold(true).Render("Entry $"),
	}
	if !compact {
		headerParts = append(headerParts,
			lipgloss.NewStyle().Width(pSL).Bold(true).Render("SL"),
			lipgloss.NewStyle().Width(pTP).Bold(true).Render("TP"),
		)
	}
	headerParts = append(headerParts, lipgloss.NewStyle().Width(pStatus).Bold(true).Render("Status"))
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

	totalCols := pID + pToken + pEntry + pStatus
	if !compact {
		totalCols += pSL + pTP
	}

	var rows []string
	rows = append(rows, headerRow)
	rows = append(rows, sepLine(totalCols))

	for _, p := range positions {
		pos, ok := p.(map[string]any)
		if !ok {
			continue
		}

		id := getString(pos, "id")
		if len(id) > 8 {
			id = id[:8]
		}

		token := getString(pos, "token")
		if token == "" {
			token = getString(pos, "symbol")
		}
		token = TruncateAddress(token)

		entryPrice := getFloat(pos, "entry_price")
		stopLoss := getFloat(pos, "stop_loss")
		takeProfit := getFloat(pos, "take_profit")
		status := getString(pos, "status")

		slStr := FormatUSD(stopLoss)
		if stopLoss == 0 {
			slStr = ui.DimStyle.Render("-")
		}
		tpStr := FormatUSD(takeProfit)
		if takeProfit == 0 {
			tpStr = ui.DimStyle.Render("-")
		}

		rowParts := []string{
			lipgloss.NewStyle().Width(pID).Foreground(ui.ColorBright).Render(id),
			lipgloss.NewStyle().Width(pToken).Render(token),
			lipgloss.NewStyle().Width(pEntry).Render(FormatUSD(entryPrice)),
		}
		if !compact {
			rowParts = append(rowParts,
				lipgloss.NewStyle().Width(pSL).Render(slStr),
				lipgloss.NewStyle().Width(pTP).Render(tpStr),
			)
		}
		rowParts = append(rowParts, lipgloss.NewStyle().Width(pStatus).Render(colorPositionStatus(status)))
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.Join(rows, "\n"),
	)

	return ui.BoxBorder.Render(content)
}

// formatOrderFailed renders a failed order action with red styling.
func formatOrderFailed(errMsg string) string {
	header := lipgloss.NewStyle().
		Foreground(ui.ColorRed).
		Bold(true).
		Render("ORDER FAILED \u2717")

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		ui.ErrorStyle.Render(errMsg),
	)

	return ui.ErrorBoxBorder.Render(content)
}

// colorStatus returns a styled status string with appropriate color.
func colorStatus(status string) string {
	lower := strings.ToLower(status)
	switch lower {
	case "pending", "open":
		return lipgloss.NewStyle().Foreground(ui.ColorGold).Render(status)
	case "filled", "executed", "active", "completed":
		return lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(status)
	case "cancelled", "canceled", "expired", "failed":
		return lipgloss.NewStyle().Foreground(ui.ColorRed).Render(status)
	case "paused":
		return lipgloss.NewStyle().Foreground(ui.ColorGold).Render(status)
	default:
		return ui.DimStyle.Render(status)
	}
}

// colorPositionStatus returns a styled position status string.
func colorPositionStatus(status string) string {
	lower := strings.ToLower(status)
	switch lower {
	case "active", "open":
		return lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(status)
	case "closing":
		return lipgloss.NewStyle().Foreground(ui.ColorGold).Render(status)
	case "closed":
		return ui.DimStyle.Render(status)
	default:
		return ui.DimStyle.Render(status)
	}
}

// formatSide returns a styled buy/sell side string.
func formatSide(side string) string {
	lower := strings.ToLower(side)
	switch lower {
	case "buy":
		return lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render(strings.ToUpper(side))
	case "sell":
		return lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true).Render(strings.ToUpper(side))
	default:
		return side
	}
}

// formatDuration converts seconds to a human-readable duration string.
func formatDuration(seconds float64) string {
	s := int(seconds)
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm", s/60)
	}
	if s < 86400 {
		h := s / 3600
		m := (s % 3600) / 60
		if m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	d := s / 86400
	h := (s % 86400) / 3600
	if h > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	return fmt.Sprintf("%dd", d)
}

// detectOrderType inspects the first order's fields to determine the type.
func detectOrderType(orders []any) string {
	if len(orders) == 0 {
		return "LIMIT"
	}
	first, ok := orders[0].(map[string]any)
	if !ok {
		return "LIMIT"
	}
	if _, has := first["total_slices"]; has {
		return "TWAP"
	}
	if _, has := first["total_intervals"]; has {
		return "DCA"
	}
	return "LIMIT"
}
