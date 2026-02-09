package formatter

import "fmt"

// unwrapData checks if the data map has a "data" wrapper and unwraps it.
// Many MCP responses return { "data": { ...actual fields... } }.
func unwrapData(dataMap map[string]any) map[string]any {
	if inner, ok := dataMap["data"].(map[string]any); ok {
		// If the inner "data" has more useful fields than the outer map,
		// use the inner one. But merge top-level fields too.
		merged := make(map[string]any)
		for k, v := range inner {
			merged[k] = v
		}
		// Carry over top-level fields that aren't in inner (like "success", "code")
		for k, v := range dataMap {
			if k == "data" {
				continue
			}
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
		}
		return merged
	}
	return dataMap
}

// FormatToolResult dispatches formatting based on the tool name. The data
// parameter is expected to be a map[string]any parsed from JSON tool
// output. Returns full multi-line rich formatted output (charts, tables, boxes).
func FormatToolResult(toolName string, data any) string {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return ""
	}

	// Unwrap { "data": { ... } } wrapper if present
	dataMap = unwrapData(dataMap)

	switch toolName {
	case "get_portfolio", "get_portfolio_summary":
		return FormatPortfolio(dataMap)
	case "get_portfolio_pnl", "get_pnl_chart":
		return FormatPnLChart(dataMap)
	case "get_token_chart", "get_token_ohlc", "get_ohlc", "get_price_chart":
		return FormatTokenChart(dataMap)
	case "search_tokens", "get_tokens_by_category", "search_token_by_slug", "get_category_tokens":
		return FormatTokenSearch(dataMap)
	case "get_token_info", "get_token_details":
		return FormatTokenInfo(dataMap)
	case "get_token_price":
		return FormatTokenPrice(dataMap)
	case "get_brewing_tokens":
		return FormatBrewingTokens(dataMap)
	case "get_swap_price", "get_swap_quote":
		return FormatSwapQuote(dataMap)
	case "execute_swap", "execute_trade":
		return FormatTradeResult(dataMap)
	case "get_trending_tokens":
		return FormatTrendingTokens(dataMap)
	// Security
	case "audit_token":
		return FormatAuditToken(dataMap)
	case "audit_tokens_batch":
		return FormatAuditBatch(dataMap)
	case "is_token_verified":
		return FormatTokenVerified(dataMap)
	// Orders
	case "create_limit_order", "create_dca_order", "create_twap_order":
		return FormatOrderCreated(dataMap)
	case "get_limit_orders":
		return FormatOrders(dataMap)
	case "get_dca_orders":
		return FormatOrders(dataMap)
	case "get_twap_orders":
		return FormatOrders(dataMap)
	case "get_limit_order", "get_dca_order", "get_twap_order", "get_position":
		return FormatOrderDetail(dataMap)
	case "cancel_limit_order", "update_limit_order",
		"pause_dca_order", "resume_dca_order", "cancel_dca_order",
		"pause_twap_order", "resume_twap_order", "cancel_twap_order":
		return FormatOrderAction(dataMap)
	case "get_positions":
		return FormatPositions(dataMap)
	// Trading
	case "get_agent_balances":
		return FormatPortfolio(dataMap)
	// Analytics
	case "get_network_stats", "get_network_volume":
		return FormatNetworkStats(dataMap)
	case "search_wallets":
		return FormatSearchWallets(dataMap)
	case "get_wallet_stats":
		return FormatWalletStats(dataMap)
	case "get_holders":
		return FormatHolders(dataMap)
	case "get_deployer_tokens":
		return FormatDeployerTokens(dataMap)
	case "get_deployer_activity":
		return FormatDeployerActivity(dataMap)
	default:
		return ""
	}
}

// FormatToolPreview returns a short one-line summary for the TUI status line.
func FormatToolPreview(toolName string, data any) string {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", data)
	}

	// Unwrap { "data": { ... } } wrapper if present
	dataMap = unwrapData(dataMap)

	switch toolName {
	case "get_portfolio", "get_portfolio_summary":
		totalValue := getFloat(dataMap, "total_value_usd")
		positions, _ := dataMap["positions"].([]any)
		if positions == nil {
			positions, _ = dataMap["tokens"].([]any)
		}
		count := getFloat(dataMap, "position_count")
		if count == 0 {
			count = float64(len(positions))
		}
		return fmt.Sprintf("Total: %s (%d positions)", FormatUSD(totalValue), int(count))

	case "get_portfolio_pnl", "get_pnl_chart":
		values := extractFloatSlice(dataMap, "chart", "data", "values", "points")
		if len(values) > 0 {
			return fmt.Sprintf("P&L chart (%d points)", len(values))
		}
		return "P&L chart loaded"

	case "get_token_chart", "get_token_ohlc", "get_ohlc", "get_price_chart":
		return "Token price chart loaded"

	case "search_tokens", "get_tokens_by_category", "search_token_by_slug", "get_category_tokens":
		tokens, _ := dataMap["tokens"].([]any)
		if tokens == nil {
			tokens, _ = dataMap["results"].([]any)
		}
		return fmt.Sprintf("%d tokens found", len(tokens))

	case "get_token_info", "get_token_details":
		name := getString(dataMap, "name")
		symbol := getString(dataMap, "symbol")
		price := getFloat(dataMap, "price_usd")
		if price == 0 {
			price = getFloat(dataMap, "price")
		}
		if name != "" {
			return fmt.Sprintf("%s (%s) %s", name, symbol, FormatUSD(price))
		}
		return "Token info loaded"

	case "get_token_price":
		prices, _ := dataMap["prices"].([]any)
		return fmt.Sprintf("%d token prices", len(prices))

	case "get_brewing_tokens":
		tokens, _ := dataMap["tokens"].([]any)
		table := getString(dataMap, "table")
		if table != "" {
			return fmt.Sprintf("%d brewing (%s)", len(tokens), table)
		}
		return fmt.Sprintf("%d brewing tokens", len(tokens))

	case "get_swap_price", "get_swap_quote":
		fromSymbol := getString(dataMap, "from_symbol")
		toSymbol := getString(dataMap, "to_symbol")
		toAmount := getFloat(dataMap, "to_amount")
		if fromSymbol != "" && toSymbol != "" {
			return fmt.Sprintf("%s -> %s %s", fromSymbol, FormatNumber(toAmount), toSymbol)
		}
		return "Swap quote ready"

	case "execute_swap", "execute_trade":
		if errMsg := getString(dataMap, "error"); errMsg != "" {
			return "Trade failed"
		}
		txHash := getString(dataMap, "tx_hash")
		if txHash == "" {
			txHash = getString(dataMap, "hash")
		}
		if txHash != "" {
			return fmt.Sprintf("Trade executed %s", TruncateAddress(txHash))
		}
		return "Trade executed"

	case "get_trending_tokens":
		tokens, _ := dataMap["tokens"].([]any)
		if tokens == nil {
			tokens, _ = dataMap["results"].([]any)
		}
		if len(tokens) > 0 {
			if top, ok := tokens[0].(map[string]any); ok {
				symbol := getString(top, "symbol")
				return fmt.Sprintf("%d trending (top: %s)", len(tokens), symbol)
			}
		}
		return fmt.Sprintf("%d trending tokens", len(tokens))

	// Security
	case "audit_token":
		risk := getString(dataMap, "risk_level")
		token := getString(dataMap, "token")
		if token != "" {
			return fmt.Sprintf("Audit: %s — Risk: %s", TruncateAddress(token), risk)
		}
		return fmt.Sprintf("Audit complete — Risk: %s", risk)

	case "audit_tokens_batch":
		audits, _ := dataMap["audits"].([]any)
		return fmt.Sprintf("%d tokens audited", len(audits))

	case "is_token_verified":
		verified, _ := getBool(dataMap, "verified")
		name := getString(dataMap, "name")
		if verified {
			return fmt.Sprintf("%s verified ✓", name)
		}
		return "Not verified"

	// Orders
	case "create_limit_order", "create_dca_order", "create_twap_order":
		msg := getString(dataMap, "message")
		if msg != "" {
			return msg
		}
		return "Order created"

	case "get_limit_orders", "get_dca_orders", "get_twap_orders":
		orders, _ := dataMap["orders"].([]any)
		return fmt.Sprintf("%d orders", len(orders))

	case "get_limit_order", "get_dca_order", "get_twap_order":
		status := getString(dataMap, "status")
		id := getString(dataMap, "id")
		if id != "" && len(id) > 8 {
			id = id[:8]
		}
		return fmt.Sprintf("Order %s — %s", id, status)

	case "cancel_limit_order", "update_limit_order",
		"pause_dca_order", "resume_dca_order", "cancel_dca_order",
		"pause_twap_order", "resume_twap_order", "cancel_twap_order":
		msg := getString(dataMap, "message")
		if msg != "" {
			return msg
		}
		return "Order updated"

	case "get_positions":
		positions, _ := dataMap["positions"].([]any)
		return fmt.Sprintf("%d positions", len(positions))

	case "get_position":
		status := getString(dataMap, "status")
		return fmt.Sprintf("Position — %s", status)

	// Trading
	case "get_agent_balances":
		totalValue := getFloat(dataMap, "total_value_usd")
		positions, _ := dataMap["positions"].([]any)
		return fmt.Sprintf("Balance: %s (%d tokens)", FormatUSD(totalValue), len(positions))

	// Analytics
	case "get_network_stats", "get_network_volume":
		summary := getString(dataMap, "summary")
		if summary != "" {
			return summary
		}
		return "Network stats loaded"

	case "search_wallets":
		wallets, _ := dataMap["wallets"].([]any)
		return fmt.Sprintf("%d wallets found", len(wallets))

	case "get_wallet_stats":
		addr := getString(dataMap, "wallet_address")
		insight := getString(dataMap, "insight")
		if insight != "" {
			return insight
		}
		if addr != "" {
			return fmt.Sprintf("Stats for %s", TruncateAddress(addr))
		}
		return "Wallet stats loaded"

	case "get_holders":
		holders, _ := dataMap["holders"].([]any)
		return fmt.Sprintf("%d holders", len(holders))

	case "get_deployer_tokens":
		tokens, _ := dataMap["tokens"].([]any)
		return fmt.Sprintf("%d deployer tokens", len(tokens))

	case "get_deployer_activity":
		activity, _ := dataMap["activity"].([]any)
		return fmt.Sprintf("%d dev trades", len(activity))

	case "get_maker_trades":
		analysis := getString(dataMap, "analysis")
		if analysis != "" {
			return analysis
		}
		return "Maker trades loaded"

	// Tracking
	case "get_kol_wallets":
		kols, _ := dataMap["kols"].([]any)
		return fmt.Sprintf("%d KOLs", len(kols))

	case "get_kol_swaps":
		swaps, _ := dataMap["swaps"].([]any)
		return fmt.Sprintf("%d KOL swaps", len(swaps))

	case "get_live_swaps":
		swaps, _ := dataMap["swaps"].([]any)
		return fmt.Sprintf("%d live swaps", len(swaps))

	case "get_user_swaps":
		swaps, _ := dataMap["swaps"].([]any)
		return fmt.Sprintf("%d user swaps", len(swaps))

	case "get_watchlist":
		watchlist, _ := dataMap["watchlist"].([]any)
		return fmt.Sprintf("%d tokens in watchlist", len(watchlist))

	case "get_streaming_status":
		ready, _ := getBool(dataMap, "ready_to_stream")
		if ready {
			return "Streaming ready ✓"
		}
		return "Streaming not connected"

	default:
		// Generic: show success/message if present
		if msg := getString(dataMap, "message"); msg != "" {
			return msg
		}
		if success, ok := getBool(dataMap, "success"); ok {
			if success {
				return "Success ✓"
			}
			return "Failed"
		}
		return fmt.Sprintf("%v", data)
	}
}
