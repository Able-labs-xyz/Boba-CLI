package ui

import "github.com/charmbracelet/lipgloss"

// Boba brand colors
var (
	ColorBoba   = lipgloss.Color("#B184F5")
	ColorDim    = lipgloss.Color("#8A5FD1")
	ColorBright = lipgloss.Color("#D4A5FF")
	ColorGold   = lipgloss.Color("#FFD700")
	ColorRed    = lipgloss.Color("#FF6B6B")
	ColorGreen  = lipgloss.Color("#50FA7B")
	ColorCyan   = lipgloss.Color("#00CED1")
	ColorPearl  = lipgloss.Color("#F5F5DC")
	ColorBrown  = lipgloss.Color("#8B4513")

	// Tool category colors
	ColorTrading   = lipgloss.Color("#4ECDC4")
	ColorPortfolio = lipgloss.Color("#9B59B6")
	ColorTokenInfo = lipgloss.Color("#F39C12")
	ColorWallet    = lipgloss.Color("#3498DB")
	ColorBrewing   = lipgloss.Color("#E74C3C")
	ColorSecurity  = lipgloss.Color("#E67E22")
	ColorOrders    = lipgloss.Color("#1ABC9C")
	ColorAnalytics = lipgloss.Color("#2ECC71")
	ColorTracking  = lipgloss.Color("#E84393")
	ColorStreaming  = lipgloss.Color("#0984E3")
)

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorBoba).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	BrightStyle = lipgloss.NewStyle().
			Foreground(ColorBright)

	GoldStyle = lipgloss.NewStyle().
			Foreground(ColorGold).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFE66D"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	// Box border style
	BoxBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBoba).
			Padding(1, 2)

	SuccessBoxBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorGreen).
				Padding(1, 2)

	ErrorBoxBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorRed).
			Padding(1, 2)

	GoldBoxBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorGold).
			Padding(1, 2)
)

// ToolColor returns the color for a tool category.
func ToolColor(toolName string) lipgloss.Color {
	switch {
	case isTrading(toolName):
		return ColorTrading
	case isPortfolio(toolName):
		return ColorPortfolio
	case isTokenTool(toolName):
		return ColorTokenInfo
	case isWallet(toolName):
		return ColorWallet
	case isBrewing(toolName):
		return ColorBrewing
	case isSecurity(toolName):
		return ColorSecurity
	case isOrders(toolName):
		return ColorOrders
	case isAnalytics(toolName):
		return ColorAnalytics
	case isTracking(toolName):
		return ColorTracking
	case isStreaming(toolName):
		return ColorStreaming
	default:
		return ColorBoba
	}
}

func isTrading(name string) bool {
	switch name {
	case "get_swap_price", "get_swap_quote", "execute_swap", "execute_trade",
		"get_agent_balances":
		return true
	}
	return false
}

func isPortfolio(name string) bool {
	switch name {
	case "get_portfolio", "get_portfolio_summary", "get_portfolio_pnl",
		"get_trade_history", "get_pnl_chart", "get_user_xp",
		"start_portfolio_stream", "get_portfolio_price_updates",
		"stop_portfolio_stream":
		return true
	}
	return false
}

func isTokenTool(name string) bool {
	switch name {
	case "get_token_info", "get_token_details", "search_tokens",
		"get_tokens_by_category", "get_trending_tokens",
		"get_token_chart", "get_token_ohlc", "get_ohlc",
		"get_price_chart", "search_token_by_slug",
		"get_token_price", "get_category_tokens":
		return true
	}
	return false
}

func isWallet(name string) bool {
	switch name {
	case "get_wallet_balance", "get_transfers", "refresh_native_balances":
		return true
	}
	return false
}

func isBrewing(name string) bool {
	switch name {
	case "get_brewing_status", "get_recent_launches",
		"get_brewing_tokens", "get_launch_feed":
		return true
	}
	return false
}

func isSecurity(name string) bool {
	switch name {
	case "audit_token", "audit_tokens_batch", "is_token_verified":
		return true
	}
	return false
}

func isOrders(name string) bool {
	switch name {
	case "create_limit_order", "get_limit_orders", "get_limit_order",
		"update_limit_order", "cancel_limit_order",
		"create_dca_order", "get_dca_orders", "get_dca_order",
		"pause_dca_order", "resume_dca_order", "cancel_dca_order",
		"create_twap_order", "get_twap_orders", "get_twap_order",
		"pause_twap_order", "resume_twap_order", "cancel_twap_order",
		"get_positions", "get_position":
		return true
	}
	return false
}

func isAnalytics(name string) bool {
	switch name {
	case "get_deployer_tokens", "get_deployer_activity",
		"get_network_volume", "get_network_stats",
		"search_wallets", "get_wallet_stats",
		"get_maker_trades", "get_holders":
		return true
	}
	return false
}

func isTracking(name string) bool {
	switch name {
	case "get_live_swaps", "get_user_swaps", "get_watchlist",
		"add_to_watchlist", "remove_from_watchlist",
		"get_kol_wallets", "get_kol_swaps", "get_kol_info",
		"check_if_kol", "get_deployer_history",
		"track_deployer", "stop_tracking_deployer",
		"add_wallet_to_tracker", "get_tracked_wallets",
		"remove_wallet_from_tracker":
		return true
	}
	return false
}

func isStreaming(name string) bool {
	switch name {
	case "stream_launches", "stream_kol_swaps",
		"stream_wallet_swaps", "stream_watchlist_swaps",
		"get_streaming_status":
		return true
	}
	return false
}

// ToolTag returns a styled category tag like [TRADE] for a tool name.
func ToolTag(toolName string) string {
	tag, color := toolTagInfo(toolName)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(color).
		Bold(true).
		Padding(0, 1).
		Render(tag)
}

func toolTagInfo(toolName string) (string, lipgloss.Color) {
	switch {
	case isTrading(toolName):
		return "TRADE", ColorTrading
	case isPortfolio(toolName):
		return "FOLIO", ColorPortfolio
	case isTokenTool(toolName):
		return "TOKEN", ColorTokenInfo
	case isWallet(toolName):
		return "WALLET", ColorWallet
	case isBrewing(toolName):
		return "BREW", ColorBrewing
	case isSecurity(toolName):
		return "AUDIT", ColorSecurity
	case isOrders(toolName):
		return "ORDER", ColorOrders
	case isAnalytics(toolName):
		return "STATS", ColorAnalytics
	case isTracking(toolName):
		return "TRACK", ColorTracking
	case isStreaming(toolName):
		return "STREAM", ColorStreaming
	default:
		return "TOOL", ColorBoba
	}
}

// Gradient helpers for richer text rendering.
var GradientPurple = []lipgloss.Color{
	lipgloss.Color("#6B3FA0"),
	lipgloss.Color("#7B52B5"),
	lipgloss.Color("#8A5FD1"),
	lipgloss.Color("#9B72E0"),
	lipgloss.Color("#B184F5"),
	lipgloss.Color("#C098FF"),
	lipgloss.Color("#D4A5FF"),
}

// RenderGradient applies a vertical gradient to text lines.
func RenderGradient(lines []string, colors []lipgloss.Color) string {
	var result []string
	for i, line := range lines {
		colorIdx := 0
		if len(colors) > 1 {
			colorIdx = i * (len(colors) - 1) / max(len(lines)-1, 1)
		}
		if colorIdx >= len(colors) {
			colorIdx = len(colors) - 1
		}
		style := lipgloss.NewStyle().Foreground(colors[colorIdx]).Bold(true)
		result = append(result, style.Render(line))
	}
	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

