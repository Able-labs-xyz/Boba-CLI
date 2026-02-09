package tui

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tradeboba/boba-cli/internal/formatter"
	"github.com/tradeboba/boba-cli/internal/proxy"
	"github.com/tradeboba/boba-cli/internal/ui"
	"github.com/tradeboba/boba-cli/internal/version"
)

type LogMsg proxy.LogEntry
type TickMsg time.Time
type BootTickMsg struct{}
type QuitStepMsg struct{}
type PortfolioMsg struct{ Data *PortfolioData }
type ChainPortfolioMsg struct{ Data *PortfolioData }
type PortfolioPollMsg struct{}

// PortfolioData holds the parsed portfolio state for the TUI panel.
type PortfolioData struct {
	TotalValueUSD    float64
	PositionValueUSD float64
	NativeValueUSD   float64
	Positions        []PortfolioPosition
	NativeBalances   []NativeBalance
	LastUpdated      time.Time
	Error            string
}

type PortfolioPosition struct {
	ChainName  string
	Symbol     string
	ValueUSD   float64
	PnlPercent float64
	PriceUSD   float64
}

type NativeBalance struct {
	ChainID    int
	ChainName  string
	Symbol     string
	Balance    float64
	BalanceUSD float64
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return TickMsg(t) })
}

func bootTick() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(_ time.Time) tea.Msg { return BootTickMsg{} })
}

func quitStep() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg { return QuitStepMsg{} })
}

func fetchPortfolio(server *proxy.ProxyServer) tea.Cmd {
	return func() tea.Msg {
		args := map[string]any{"user_id": "me"}
		respBody, err := server.CallTool("get_portfolio", args)
		if err != nil {
			return PortfolioMsg{Data: &PortfolioData{
				Error:       err.Error(),
				LastUpdated: time.Now(),
			}}
		}

		var raw map[string]any
		if err := json.Unmarshal(respBody, &raw); err != nil {
			return PortfolioMsg{Data: &PortfolioData{
				Error:       "failed to parse portfolio data",
				LastUpdated: time.Now(),
			}}
		}

		data := &PortfolioData{
			TotalValueUSD:    parseFloat(raw, "total_value_usd"),
			PositionValueUSD: parseFloat(raw, "position_value_usd"),
			NativeValueUSD:   parseFloat(raw, "native_value_usd"),
			LastUpdated:      time.Now(),
		}

		// Parse positions
		if positions, ok := raw["positions"].([]any); ok {
			for _, p := range positions {
				pos, ok := p.(map[string]any)
				if !ok {
					continue
				}
				chainName := parseString(pos, "chain_name")
				if chainName == "" {
					chainName = parseString(pos, "chain")
				}
				if chainName == "" {
					chainName = parseString(pos, "chainName")
				}
				if chainName == "" {
					chainName = parseString(pos, "network")
				}
				data.Positions = append(data.Positions, PortfolioPosition{
					ChainName:  chainName,
					Symbol:     parseString(pos, "symbol"),
					ValueUSD:   parseFloat(pos, "value_usd"),
					PnlPercent: parseFloat(pos, "pnl_percent"),
					PriceUSD:   parseFloat(pos, "price_usd"),
				})
			}
			// Sort by value descending
			sort.Slice(data.Positions, func(i, j int) bool {
				return data.Positions[i].ValueUSD > data.Positions[j].ValueUSD
			})
		}

		// Parse native balances
		if balances, ok := raw["native_balances"].([]any); ok {
			for _, b := range balances {
				bal, ok := b.(map[string]any)
				if !ok {
					continue
				}
				data.NativeBalances = append(data.NativeBalances, NativeBalance{
					ChainID:    int(parseFloat(bal, "chain_id")),
					ChainName:  parseString(bal, "chain_name"),
					Symbol:     parseString(bal, "symbol"),
					Balance:    parseFloat(bal, "balance"),
					BalanceUSD: parseFloat(bal, "balance_usd"),
				})
			}
		}

		return PortfolioMsg{Data: data}
	}
}

// fetchChainPortfolio fetches portfolio data filtered to a specific chain.
// The MCP get_portfolio tool accepts a "chain" string slug (e.g. "solana", "eth").
func fetchChainPortfolio(server *proxy.ProxyServer, chainSlug string) tea.Cmd {
	return func() tea.Msg {
		args := map[string]any{
			"user_id": "me",
			"chain":   chainSlug,
		}
		respBody, err := server.CallTool("get_portfolio", args)
		if err != nil {
			return ChainPortfolioMsg{Data: &PortfolioData{
				Error:       err.Error(),
				LastUpdated: time.Now(),
			}}
		}

		var raw map[string]any
		if err := json.Unmarshal(respBody, &raw); err != nil {
			return ChainPortfolioMsg{Data: &PortfolioData{
				Error:       "failed to parse chain portfolio",
				LastUpdated: time.Now(),
			}}
		}

		data := &PortfolioData{
			TotalValueUSD:    parseFloat(raw, "total_value_usd"),
			PositionValueUSD: parseFloat(raw, "position_value_usd"),
			NativeValueUSD:   parseFloat(raw, "native_value_usd"),
			LastUpdated:      time.Now(),
		}

		if positions, ok := raw["positions"].([]any); ok {
			for _, p := range positions {
				pos, ok := p.(map[string]any)
				if !ok {
					continue
				}
				data.Positions = append(data.Positions, PortfolioPosition{
					ChainName:  parseString(pos, "chain_name"),
					Symbol:     parseString(pos, "symbol"),
					ValueUSD:   parseFloat(pos, "value_usd"),
					PnlPercent: parseFloat(pos, "pnl_percent"),
					PriceUSD:   parseFloat(pos, "price_usd"),
				})
			}
			sort.Slice(data.Positions, func(i, j int) bool {
				return data.Positions[i].ValueUSD > data.Positions[j].ValueUSD
			})
		}

		if balances, ok := raw["native_balances"].([]any); ok {
			for _, b := range balances {
				bal, ok := b.(map[string]any)
				if !ok {
					continue
				}
				data.NativeBalances = append(data.NativeBalances, NativeBalance{
					ChainID:    int(parseFloat(bal, "chain_id")),
					ChainName:  parseString(bal, "chain_name"),
					Symbol:     parseString(bal, "symbol"),
					Balance:    parseFloat(bal, "balance"),
					BalanceUSD: parseFloat(bal, "balance_usd"),
				})
			}
		}

		return ChainPortfolioMsg{Data: data}
	}
}

// parseFloat safely extracts a float64 from a map, handling string values.
func parseFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
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

// parseString safely extracts a string from a map.
func parseString(m map[string]any, key string) string {
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

var bootStepLabels = []string{
	"Generating session token",
	"Binding to port",
	"Authenticating agent",
	"Syncing tool manifest",
	"Proxy online",
}

var quitSteps = []string{
	"Clearing session token...",
	"Stopping proxy...",
	"Goodbye!",
}

type toolTag struct {
	label string
	color lipgloss.Color
}

var toolCategoryMap = map[string]toolTag{
	// Trading
	"get_swap_price":      {label: "TRADE", color: ui.ColorTrading},
	"get_swap_quote":      {label: "TRADE", color: ui.ColorTrading},
	"execute_swap":        {label: "TRADE", color: ui.ColorTrading},
	"execute_trade":       {label: "TRADE", color: ui.ColorTrading},
	"get_agent_balances":  {label: "TRADE", color: ui.ColorTrading},
	// Portfolio
	"get_portfolio":              {label: "FOLIO", color: ui.ColorPortfolio},
	"get_portfolio_summary":      {label: "FOLIO", color: ui.ColorPortfolio},
	"get_portfolio_pnl":          {label: "FOLIO", color: ui.ColorPortfolio},
	"get_trade_history":          {label: "FOLIO", color: ui.ColorPortfolio},
	"get_pnl_chart":              {label: "FOLIO", color: ui.ColorPortfolio},
	"get_user_xp":                {label: "FOLIO", color: ui.ColorPortfolio},
	"start_portfolio_stream":     {label: "FOLIO", color: ui.ColorPortfolio},
	"get_portfolio_price_updates": {label: "FOLIO", color: ui.ColorPortfolio},
	"stop_portfolio_stream":      {label: "FOLIO", color: ui.ColorPortfolio},
	// Token
	"get_token_info":         {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_token_details":      {label: "TOKEN", color: ui.ColorTokenInfo},
	"search_tokens":          {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_tokens_by_category": {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_trending_tokens":    {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_token_chart":        {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_token_ohlc":         {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_ohlc":               {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_price_chart":        {label: "TOKEN", color: ui.ColorTokenInfo},
	"search_token_by_slug":   {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_token_price":        {label: "TOKEN", color: ui.ColorTokenInfo},
	"get_category_tokens":    {label: "TOKEN", color: ui.ColorTokenInfo},
	// Wallet
	"get_wallet_balance":      {label: "WALLET", color: ui.ColorWallet},
	"get_transfers":           {label: "WALLET", color: ui.ColorWallet},
	"refresh_native_balances": {label: "WALLET", color: ui.ColorWallet},
	// Brewing
	"get_brewing_status":  {label: "BREW", color: ui.ColorBrewing},
	"get_recent_launches": {label: "BREW", color: ui.ColorBrewing},
	"get_brewing_tokens":  {label: "BREW", color: ui.ColorBrewing},
	"get_launch_feed":     {label: "BREW", color: ui.ColorBrewing},
	// Security
	"audit_token":        {label: "AUDIT", color: ui.ColorSecurity},
	"audit_tokens_batch": {label: "AUDIT", color: ui.ColorSecurity},
	"is_token_verified":  {label: "AUDIT", color: ui.ColorSecurity},
	// Orders
	"create_limit_order":  {label: "ORDER", color: ui.ColorOrders},
	"get_limit_orders":    {label: "ORDER", color: ui.ColorOrders},
	"get_limit_order":     {label: "ORDER", color: ui.ColorOrders},
	"update_limit_order":  {label: "ORDER", color: ui.ColorOrders},
	"cancel_limit_order":  {label: "ORDER", color: ui.ColorOrders},
	"create_dca_order":    {label: "ORDER", color: ui.ColorOrders},
	"get_dca_orders":      {label: "ORDER", color: ui.ColorOrders},
	"get_dca_order":       {label: "ORDER", color: ui.ColorOrders},
	"pause_dca_order":     {label: "ORDER", color: ui.ColorOrders},
	"resume_dca_order":    {label: "ORDER", color: ui.ColorOrders},
	"cancel_dca_order":    {label: "ORDER", color: ui.ColorOrders},
	"create_twap_order":   {label: "ORDER", color: ui.ColorOrders},
	"get_twap_orders":     {label: "ORDER", color: ui.ColorOrders},
	"get_twap_order":      {label: "ORDER", color: ui.ColorOrders},
	"pause_twap_order":    {label: "ORDER", color: ui.ColorOrders},
	"resume_twap_order":   {label: "ORDER", color: ui.ColorOrders},
	"cancel_twap_order":   {label: "ORDER", color: ui.ColorOrders},
	"get_positions":       {label: "ORDER", color: ui.ColorOrders},
	"get_position":        {label: "ORDER", color: ui.ColorOrders},
	// Analytics
	"get_deployer_tokens":   {label: "STATS", color: ui.ColorAnalytics},
	"get_deployer_activity": {label: "STATS", color: ui.ColorAnalytics},
	"get_network_volume":    {label: "STATS", color: ui.ColorAnalytics},
	"get_network_stats":     {label: "STATS", color: ui.ColorAnalytics},
	"search_wallets":        {label: "STATS", color: ui.ColorAnalytics},
	"get_wallet_stats":      {label: "STATS", color: ui.ColorAnalytics},
	"get_maker_trades":      {label: "STATS", color: ui.ColorAnalytics},
	"get_holders":           {label: "STATS", color: ui.ColorAnalytics},
	// Tracking
	"get_live_swaps":             {label: "TRACK", color: ui.ColorTracking},
	"get_user_swaps":             {label: "TRACK", color: ui.ColorTracking},
	"get_watchlist":              {label: "TRACK", color: ui.ColorTracking},
	"add_to_watchlist":           {label: "TRACK", color: ui.ColorTracking},
	"remove_from_watchlist":      {label: "TRACK", color: ui.ColorTracking},
	"get_kol_wallets":            {label: "TRACK", color: ui.ColorTracking},
	"get_kol_swaps":              {label: "TRACK", color: ui.ColorTracking},
	"get_kol_info":               {label: "TRACK", color: ui.ColorTracking},
	"check_if_kol":               {label: "TRACK", color: ui.ColorTracking},
	"get_deployer_history":       {label: "TRACK", color: ui.ColorTracking},
	"track_deployer":             {label: "TRACK", color: ui.ColorTracking},
	"stop_tracking_deployer":     {label: "TRACK", color: ui.ColorTracking},
	"add_wallet_to_tracker":      {label: "TRACK", color: ui.ColorTracking},
	"get_tracked_wallets":        {label: "TRACK", color: ui.ColorTracking},
	"remove_wallet_from_tracker": {label: "TRACK", color: ui.ColorTracking},
	// Streaming
	"stream_launches":        {label: "STREAM", color: ui.ColorStreaming},
	"stream_kol_swaps":       {label: "STREAM", color: ui.ColorStreaming},
	"stream_wallet_swaps":    {label: "STREAM", color: ui.ColorStreaming},
	"stream_watchlist_swaps": {label: "STREAM", color: ui.ColorStreaming},
	"get_streaming_status":   {label: "STREAM", color: ui.ColorStreaming},
}

var defaultTag = toolTag{label: "TOOL", color: ui.ColorBoba}

// chainOrder defines the fixed display order for chain tabs.
var chainOrder = []string{
	"Solana", "Base", "BSC", "Ethereum", "Arbitrum",
	"Avalanche", "Ape Chain", "HyperEVM", "Monad",
}

// chainNameToSlug maps display chain names to the MCP tool's chain parameter slugs.
// The MCP get_portfolio tool accepts these string slugs (not numeric chain IDs).
var chainNameToSlug = map[string]string{
	"Solana":    "solana",
	"Ethereum":  "eth",
	"Ape Chain": "apechain",
	"BSC":       "bsc",
	"Avalanche": "avax",
	"Base":      "base",
	"Arbitrum":  "arb",
	"HyperEVM":  "hyperevm",
	"Monad":     "monad",
}

func getToolTag(tool string) toolTag {
	if t, ok := toolCategoryMap[tool]; ok {
		return t
	}
	return defaultTag
}

// toolDescriptions maps tool names to human-readable descriptions shown while
// a request is pending.
var toolDescriptions = map[string]string{
	// Trading
	"get_swap_price":     "Getting swap quote...",
	"get_swap_quote":     "Getting swap quote...",
	"execute_swap":       "Executing trade...",
	"execute_trade":      "Executing trade...",
	"get_agent_balances": "Fetching balances...",
	// Portfolio
	"get_portfolio":         "Fetching portfolio...",
	"get_portfolio_summary": "Fetching portfolio...",
	"get_portfolio_pnl":     "Getting P&L data...",
	"get_pnl_chart":         "Loading P&L chart...",
	// Token
	"get_token_info":         "Looking up token...",
	"get_token_details":      "Looking up token...",
	"search_tokens":          "Searching tokens...",
	"get_tokens_by_category": "Searching tokens...",
	"get_trending_tokens":    "Getting trending...",
	"get_token_chart":        "Loading price chart...",
	"get_token_ohlc":         "Loading price chart...",
	"get_ohlc":               "Loading price chart...",
	"get_price_chart":        "Loading price chart...",
	"search_token_by_slug":   "Searching tokens...",
	"get_token_price":        "Getting token price...",
	"get_category_tokens":    "Searching by category...",
	// Wallet
	"get_wallet_balance": "Checking wallet...",
	// Brewing
	"get_brewing_tokens":  "Getting brewing tokens...",
	"get_brewing_status":  "Checking new launches...",
	"get_recent_launches": "Getting recent launches...",
	"get_launch_feed":     "Loading launch feed...",
	// Security
	"audit_token":        "Auditing token...",
	"audit_tokens_batch": "Auditing tokens...",
	"is_token_verified":  "Checking verification...",
	// Orders
	"create_limit_order": "Creating limit order...",
	"get_limit_orders":   "Getting limit orders...",
	"get_limit_order":    "Getting order detail...",
	"update_limit_order": "Updating order...",
	"cancel_limit_order": "Cancelling order...",
	"create_dca_order":   "Creating DCA order...",
	"get_dca_orders":     "Getting DCA orders...",
	"get_dca_order":      "Getting DCA detail...",
	"pause_dca_order":    "Pausing DCA...",
	"resume_dca_order":   "Resuming DCA...",
	"cancel_dca_order":   "Cancelling DCA...",
	"create_twap_order":  "Creating TWAP order...",
	"get_twap_orders":    "Getting TWAP orders...",
	"get_twap_order":     "Getting TWAP detail...",
	"pause_twap_order":   "Pausing TWAP...",
	"resume_twap_order":  "Resuming TWAP...",
	"cancel_twap_order":  "Cancelling TWAP...",
	"get_positions":      "Getting positions...",
	"get_position":       "Getting position...",
	// Analytics
	"get_deployer_tokens":   "Getting deployer tokens...",
	"get_deployer_activity": "Getting dev activity...",
	"get_network_volume":    "Getting network volume...",
	"get_network_stats":     "Getting network stats...",
	"search_wallets":        "Searching wallets...",
	"get_wallet_stats":      "Getting wallet stats...",
	"get_maker_trades":      "Getting maker trades...",
	"get_holders":           "Getting holders...",
	// Tracking
	"get_live_swaps":             "Getting live swaps...",
	"get_user_swaps":             "Getting user swaps...",
	"get_watchlist":              "Getting watchlist...",
	"add_to_watchlist":           "Adding to watchlist...",
	"remove_from_watchlist":      "Removing from watchlist...",
	"get_kol_wallets":            "Getting KOL wallets...",
	"get_kol_swaps":              "Getting KOL swaps...",
	"get_kol_info":               "Getting KOL info...",
	"check_if_kol":               "Checking KOL...",
	"get_deployer_history":       "Getting dev history...",
	"track_deployer":             "Starting tracker...",
	"stop_tracking_deployer":     "Stopping tracker...",
	"add_wallet_to_tracker":      "Adding wallet...",
	"get_tracked_wallets":        "Getting tracked wallets...",
	"remove_wallet_from_tracker": "Removing wallet...",
	// Streaming
	"stream_launches":        "Streaming launches...",
	"stream_kol_swaps":       "Streaming KOL swaps...",
	"stream_wallet_swaps":    "Streaming wallet...",
	"stream_watchlist_swaps": "Streaming watchlist...",
	"get_streaming_status":   "Checking streams...",
}

type ProxyViewModel struct {
	logo      string
	agentName string
	evmAddr   string
	solAddr   string
	port      int

	logEntries []proxy.LogEntry
	viewport   viewport.Model
	spinner    spinner.Model
	server     *proxy.ProxyServer
	autoScroll bool

	activeTab             int
	tabs                  []string
	chainSlugs            map[string]string
	chainPortfolio        *PortfolioData
	chainPortfolioLoading bool

	startTime    time.Time
	requestCount int
	errorCount   int

	portfolio        *PortfolioData
	portfolioLoading bool
	portfolioFlash   int

	showConfig bool

	// phases: "boot" -> "running" -> "quitting"
	phase string

	bootStep     int
	bootFrame    int
	bootGlitch   int
	bootProgress progress.Model

	quitStep int

	idleFrame int

	width  int
	height int
	ready  bool
}

func NewProxyViewModel(server *proxy.ProxyServer, agentName, evmAddr, solAddr string, port int) ProxyViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorBoba)

	prog := progress.New(
		progress.WithGradient(string(ui.ColorBoba), "#333333"),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return ProxyViewModel{
		logo:         ui.RenderLogo(),
		autoScroll:   true,
		agentName:    agentName,
		evmAddr:      evmAddr,
		solAddr:      solAddr,
		port:         port,
		spinner:      s,
		server:       server,
		startTime:    time.Now(),
		phase:        "boot",
		bootStep:     0,
		bootFrame:    0,
		bootProgress: prog,
		tabs:       []string{"All"},
		activeTab:  0,
		chainSlugs: make(map[string]string),
	}
}

func (m ProxyViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		bootTick(),
	)
}

func (m ProxyViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "q", "ctrl+c":
			if m.phase == "boot" {
				return m, tea.Quit
			}
			if m.phase == "quitting" {
				return m, nil
			}
			m.phase = "quitting"
			m.quitStep = 0
			return m, quitStep()
		case "tab", "right":
			if m.phase == "running" && len(m.tabs) > 1 {
				prevTab := m.activeTab
				m.activeTab++
				if m.activeTab >= len(m.tabs) {
					m.activeTab = len(m.tabs) - 1
				}
				if m.activeTab != prevTab {
					m.recalcViewport()
					if m.activeTab > 0 {
						chainName := m.tabs[m.activeTab]
						if slug, ok := m.chainSlugs[chainName]; ok {
							m.chainPortfolioLoading = true
							m.chainPortfolio = nil
							return m, fetchChainPortfolio(m.server, slug)
						}
					}
				}
			}
		case "shift+tab", "left":
			if m.phase == "running" && m.activeTab > 0 {
				m.activeTab--
				m.recalcViewport()
				if m.activeTab > 0 {
					chainName := m.tabs[m.activeTab]
					if slug, ok := m.chainSlugs[chainName]; ok {
						m.chainPortfolioLoading = true
						m.chainPortfolio = nil
						return m, fetchChainPortfolio(m.server, slug)
					}
				}
			}
		case "c":
			if m.phase == "running" {
				m.showConfig = !m.showConfig
				m.recalcViewport()
			}
		case "up", "k", "pgup":
			if m.phase == "running" {
				m.autoScroll = false
			}
		case "end", "G":
			if m.phase == "running" && m.ready {
				m.autoScroll = true
				m.viewport.GotoBottom()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		formatter.TermWidth = msg.Width
		// boot phase doesn't need resize handling
		if m.phase == "running" {
			m.recalcViewport()
		}

	// -- boot sequence tick (40ms per frame) --------------------------------
	case BootTickMsg:
		if m.phase != "boot" {
			break
		}
		m.bootFrame++
		m.bootGlitch++

		// Advance steps rapidly (every ~0.3s = every 7 frames)
		stepFrames := []int{5, 12, 19, 26, 33}
		for i, f := range stepFrames {
			if m.bootFrame == f && i < len(bootStepLabels) {
				m.bootStep = i + 1
				m.bootGlitch = 0
			}
		}

		// Boot complete at frame 40 (~1.6s) then transition
		if m.bootFrame >= 40 {
			m.phase = "running"
			m.startTime = time.Now()
			m.portfolioLoading = true
			m.recalcViewport()
			return m, tea.Batch(
				tickEvery(time.Second),
				listenForLogs(m.server.LogChannel()),
				fetchPortfolio(m.server),
			)
		}
		return m, bootTick()

	// -- quit sequence tick ------------------------------------------------
	case QuitStepMsg:
		if m.phase != "quitting" {
			break
		}
		m.quitStep++
		if m.quitStep >= len(quitSteps) {
			return m, tea.Quit
		}
		return m, quitStep()

	// -- portfolio data received -------------------------------------------
	case PortfolioMsg:
		m.portfolio = msg.Data
		m.portfolioLoading = false
		m.portfolioFlash = 3 // flash for 3 ticks after refresh
		// Build dynamic tabs from portfolio data
		m.buildTabs()
		if m.phase == "running" {
			m.recalcViewport()
		}
		// Schedule next poll in 30 seconds
		cmds = append(cmds, tea.Tick(30*time.Second, func(_ time.Time) tea.Msg {
			return PortfolioPollMsg{}
		}))

	// -- chain-specific portfolio data received ----------------------------
	case ChainPortfolioMsg:
		m.chainPortfolio = msg.Data
		m.chainPortfolioLoading = false
		if m.phase == "running" {
			m.recalcViewport()
		}

	// -- portfolio poll timer fired ----------------------------------------
	case PortfolioPollMsg:
		if m.phase == "running" {
			m.portfolioLoading = true
			cmds = append(cmds, fetchPortfolio(m.server))
		}

	// -- 1-second heartbeat ------------------------------------------------
	case TickMsg:
		if m.phase == "running" {
			m.idleFrame++
			if m.portfolioFlash > 0 {
				m.portfolioFlash--
			}
			if m.ready {
				m.viewport.SetContent(m.renderViewportContent())
			}
		}
		cmds = append(cmds, tickEvery(time.Second))

	// -- proxy log entry ---------------------------------------------------
	case LogMsg:
		entry := proxy.LogEntry(msg)
		m.logEntries = append(m.logEntries, entry)
		if entry.Status == "success" {
			m.requestCount++
		}
		if entry.Status == "error" {
			m.errorCount++
		}
		if m.ready {
			m.viewport.SetContent(m.renderViewportContent())
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
		}
		cmds = append(cmds, listenForLogs(m.server.LogChannel()))

	// -- spinner -----------------------------------------------------------
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	// -- progress bar animation --------------------------------------------
	case progress.FrameMsg:
		progressModel, cmd := m.bootProgress.Update(msg)
		m.bootProgress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	// viewport passthrough
	if m.ready && m.phase == "running" {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		if m.viewport.AtBottom() {
			m.autoScroll = true
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ProxyViewModel) recalcViewport() {
	portfolioHeight := m.portfolioPanelHeight()
	if portfolioHeight > 0 {
		portfolioHeight++ // +1 for the "\n" after the panel
	}

	configHeight := 0
	if m.showConfig {
		configHeight = m.configPanelHeight() + 1 // +1 for "\n" after panel
	}

	headerHeight := 1 + // compact logo line
		1 + // blank after logo
		2 + // tab bar (tabs + border)
		portfolioHeight +
		configHeight +
		1 + // stats bar
		1 + // blank
		1 + // spec line
		1 + // blank
		1 + // activity header
		1 // separator
	footerHeight := 2 // footer separator + hint line

	vpHeight := m.height - headerHeight - footerHeight
	if vpHeight < 3 {
		vpHeight = 3
	}

	if !m.ready {
		m.viewport = viewport.New(m.width, vpHeight)
		m.viewport.Style = lipgloss.NewStyle()
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
	}
	m.viewport.SetContent(m.renderViewportContent())
}

// portfolioPanelHeight returns the number of terminal lines the portfolio panel
// will occupy, including borders. Accounts for active tab.
func (m *ProxyViewModel) portfolioPanelHeight() int {
	if m.portfolio == nil && !m.portfolioLoading {
		return 0
	}
	if m.portfolio == nil || m.portfolio.Error != "" {
		return 3 // border top + content + border bottom
	}

	if m.activeTab == 0 {
		// "All" tab: compact panel
		contentLines := 2 // header + blank
		nativeCount := len(m.portfolio.NativeBalances)
		if nativeCount > 0 {
			contentLines += nativeCount
			contentLines++ // blank after natives
		}
		posCount := len(m.portfolio.Positions)
		if posCount == 0 {
			contentLines++
		} else {
			shown := posCount
			if shown > 4 {
				shown = 4
			}
			contentLines += shown
			if posCount > 4 {
				contentLines++
			}
		}
		return contentLines + 2 // +2 for borders
	}

	// Chain tab: uses chainPortfolio data
	if m.chainPortfolio == nil || m.chainPortfolioLoading {
		return 3 // loading state: border + content + border
	}
	if m.chainPortfolio.Error != "" {
		return 3
	}

	contentLines := 2 // header + blank
	nativeCount := len(m.chainPortfolio.NativeBalances)
	if nativeCount > 0 {
		contentLines += nativeCount
		contentLines++ // blank after natives
	}
	posCount := len(m.chainPortfolio.Positions)
	if posCount == 0 {
		contentLines++
	} else {
		contentLines += posCount
	}

	return contentLines + 2 // +2 for borders
}


func (m ProxyViewModel) View() string {
	switch m.phase {
	case "boot":
		return m.viewBoot()
	case "quitting":
		return m.viewQuit()
	default:
		return m.viewRunning()
	}
}

var bootBubbleChars = []string{".", "o", "O", "◯", "●", "◉"}

func (m ProxyViewModel) viewBoot() string {
	var b strings.Builder

	// ---- Bubble field — particles rising upward ----
	fieldH := 8
	fieldW := m.width
	if fieldW < 40 {
		fieldW = 40
	}

	grid := make([][]rune, fieldH)
	for y := range grid {
		grid[y] = make([]rune, fieldW)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	numBubbles := fieldW / 2
	if numBubbles < 40 {
		numBubbles = 40
	}
	for i := range numBubbles {
		seed := i*7 + 13
		baseX := (seed*31 + i*17) % fieldW
		speed := 1 + (seed % 4)
		cycleLen := fieldH + 14
		baseY := fieldH + 5 - (m.bootFrame*speed/3+i*3)%cycleLen
		wobble := int(math.Sin(float64(m.bootFrame)*0.15+float64(i)*2.3) * 1.5)
		x := baseX + wobble
		y := baseY

		if y >= 0 && y < fieldH && x >= 0 && x < fieldW {
			charIdx := y * len(bootBubbleChars) / fieldH
			if charIdx >= len(bootBubbleChars) {
				charIdx = len(bootBubbleChars) - 1
			}
			ch := bootBubbleChars[charIdx]
			for _, r := range ch {
				grid[y][x] = r
				break
			}
		}
	}

	gradColors := []lipgloss.Color{
		lipgloss.Color("#3B1F6E"),
		lipgloss.Color("#4B2D8E"),
		lipgloss.Color("#6B3FA0"),
		lipgloss.Color("#7B52B5"),
		lipgloss.Color("#8A5FD1"),
		lipgloss.Color("#9B72E0"),
		lipgloss.Color("#B184F5"),
		lipgloss.Color("#D4A5FF"),
	}
	for y := range fieldH {
		colorIdx := y * (len(gradColors) - 1) / max(fieldH-1, 1)
		if colorIdx >= len(gradColors) {
			colorIdx = len(gradColors) - 1
		}
		style := lipgloss.NewStyle().Foreground(gradColors[colorIdx])
		b.WriteString(style.Render(string(grid[y])))
		b.WriteString("\n")
	}

	// ---- Logo fading in from block characters ----
	b.WriteString("\n")
	logoProgress := float64(m.bootFrame-2) / 18.0
	if logoProgress < 0 {
		logoProgress = 0
	}
	if logoProgress > 1.0 {
		logoProgress = 1.0
	}

	logoLines := ui.LogoLines()
	for i, line := range logoLines {
		lineP := logoProgress*1.3 - float64(i)*0.03
		if lineP < 0 {
			lineP = 0
		}
		if lineP > 1.0 {
			lineP = 1.0
		}

		var result string
		if lineP >= 1.0 {
			color := ui.ColorBoba
			if i < len(ui.GradientPurple) {
				color = ui.GradientPurple[i]
			}
			result = lipgloss.NewStyle().Foreground(color).Bold(true).Render(line)
		} else if lineP > 0 {
			result = bootPartialReveal(line, lineP, i)
		} else {
			result = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a1a2e")).Render(line)
		}
		b.WriteString(result + "\n")
	}

	b.WriteString("\n")

	// ---- Boot steps with typing reveal ----
	checkStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
	activeStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#222222"))

	for i, label := range bootStepLabels {
		if i < m.bootStep {
			fmt.Fprintf(&b, "  %s  %s\n",
				checkStyle.Render("●"),
				doneStyle.Render(label))
		} else if i == m.bootStep && m.bootStep < len(bootStepLabels) {
			charsRevealed := m.bootGlitch
			if charsRevealed > len(label) {
				charsRevealed = len(label)
			}
			revealed := label[:charsRevealed]
			cursor := ""
			if charsRevealed < len(label) {
				cursor = "█"
			}
			fmt.Fprintf(&b, "  %s  %s%s\n",
				m.spinner.View(),
				activeStyle.Render(revealed),
				lipgloss.NewStyle().Foreground(ui.ColorBoba).Render(cursor))
		} else {
			fmt.Fprintf(&b, "       %s\n",
				pendingStyle.Render(strings.Repeat("·", len(label))))
		}
	}

	// ---- Progress bar ----
	b.WriteString("\n")
	pct := float64(m.bootStep) / float64(len(bootStepLabels))
	b.WriteString("  ")
	b.WriteString(m.bootProgress.ViewAs(pct))
	b.WriteString("\n")

	// ---- "CONNECTED" badge at the end ----
	if m.bootFrame >= 36 {
		onlineStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorBoba).
			Bold(true).
			Padding(0, 2)
		b.WriteString("\n  " + onlineStyle.Render(" CONNECTED "))
		b.WriteString("\n")
	}

	return b.String()
}

// bootPartialReveal resolves logo characters left-to-right with block chars for unresolved.
func bootPartialReveal(line string, progress float64, lineIdx int) string {
	runes := []rune(line)
	totalNonSpace := 0
	for _, r := range runes {
		if r != ' ' {
			totalNonSpace++
		}
	}
	resolved := int(float64(totalNonSpace) * progress)

	var result strings.Builder
	nonSpaceIdx := 0
	blockReplace := []rune{'░', '▒', '▓', '█'}

	color := ui.ColorBoba
	if lineIdx < len(ui.GradientPurple) {
		color = ui.GradientPurple[lineIdx]
	}
	resolvedStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	unresolvedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333355"))

	for _, r := range runes {
		if r == ' ' {
			result.WriteRune(' ')
			continue
		}
		if nonSpaceIdx < resolved {
			result.WriteString(resolvedStyle.Render(string(r)))
		} else {
			ch := blockReplace[rand.Intn(len(blockReplace))]
			result.WriteString(unresolvedStyle.Render(string(ch)))
		}
		nonSpaceIdx++
	}
	return result.String()
}

func (m ProxyViewModel) viewQuit() string {
	var b strings.Builder

	b.WriteString("\n")

	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)
	b.WriteString(titleStyle.Render("  SHUTTING DOWN"))
	b.WriteString("\n\n")

	checkStyle := lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(ui.ColorBright)
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)

	for i, step := range quitSteps {
		if i < m.quitStep {
			b.WriteString(fmt.Sprintf("  %s  %s\n", checkStyle.Render("[ok]"), lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(step)))
		} else if i == m.quitStep {
			b.WriteString(fmt.Sprintf("  %s  %s\n", m.spinner.View(), activeStyle.Render(step)))
		} else {
			b.WriteString(fmt.Sprintf("       %s\n", dimStyle.Render(step)))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func (m ProxyViewModel) viewRunning() string {
	var b strings.Builder

	verStyle := lipgloss.NewStyle().Foreground(ui.ColorGold)
	b.WriteString("  " + ui.RenderLogoCompact() + "  " + verStyle.Render(version.Version))
	b.WriteString("\n\n")

	b.WriteString(m.renderTabBar())
	b.WriteString("\n")

	if m.portfolio != nil || m.portfolioLoading {
		if m.activeTab == 0 {
			b.WriteString(m.renderPortfolioPanel())
		} else if m.activeTab < len(m.tabs) {
			b.WriteString(m.renderChainPortfolio(m.tabs[m.activeTab]))
		}
		b.WriteString("\n")
	}

	if m.showConfig {
		b.WriteString(m.renderConfigPanel())
		b.WriteString("\n")
	}

	b.WriteString(m.renderStatsBar())
	b.WriteString("\n\n")

	b.WriteString(m.renderSpecLine())
	b.WriteString("\n")

	headerStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)

	var badge string
	if m.autoScroll {
		badge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a2e")).
			Background(ui.ColorGreen).
			Bold(true).
			Padding(0, 1).
			Render("LIVE")
	} else {
		totalLines := strings.Count(m.viewport.View(), "\n") + 1
		currentLine := m.viewport.YOffset + m.viewport.Height
		if currentLine > totalLines {
			currentLine = totalLines
		}
		badge = lipgloss.NewStyle().
			Foreground(ui.ColorCyan).
			Bold(true).
			Render(fmt.Sprintf("[%d/%d]", currentLine, len(m.logEntries)))
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n", headerStyle.Render("ACTIVITY LOG"), badge))

	// Separator width
	sepLen := 50
	if m.width > 4 {
		sepLen = m.width - 4
	}
	if sepLen > 80 {
		sepLen = 80
	}

	sepStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
	b.WriteString(sepStyle.Render("  " + strings.Repeat("━", sepLen)))
	b.WriteString("\n")

	if m.ready {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(m.renderIdleText())
	}

	b.WriteString("\n")
	footerSep := lipgloss.NewStyle().Foreground(ui.ColorDim)
	hintDim := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	hintKey := lipgloss.NewStyle().Foreground(ui.ColorBoba)
	b.WriteString(footerSep.Render("  " + strings.Repeat("━", sepLen)))
	b.WriteString("\n")
	b.WriteString(hintDim.Render("  ") +
		hintKey.Render("q") + hintDim.Render(" quit  ") +
		hintKey.Render("←→") + hintDim.Render(" tabs  ") +
		hintKey.Render("↑↓") + hintDim.Render(" scroll  ") +
		hintKey.Render("end") + hintDim.Render(" follow  ") +
		hintKey.Render("c") + hintDim.Render(" config"))

	return b.String()
}

// buildTabs rebuilds the tab list from the current portfolio data using the fixed chain order.
func (m *ProxyViewModel) buildTabs() {
	if m.portfolio == nil || m.portfolio.Error != "" {
		m.tabs = []string{"All"}
		if m.activeTab >= len(m.tabs) {
			m.activeTab = 0
		}
		return
	}

	// Collect chains present in portfolio data
	present := make(map[string]bool)
	for _, nb := range m.portfolio.NativeBalances {
		if nb.ChainName != "" {
			present[nb.ChainName] = true
		}
	}
	for _, pos := range m.portfolio.Positions {
		if pos.ChainName != "" {
			present[pos.ChainName] = true
		}
	}

	// Build tabs in fixed order, only including chains that are present
	m.chainSlugs = make(map[string]string)
	var chainNames []string
	for _, name := range chainOrder {
		if present[name] {
			chainNames = append(chainNames, name)
			if slug, ok := chainNameToSlug[name]; ok {
				m.chainSlugs[name] = slug
			} else {
				m.chainSlugs[name] = strings.ToLower(name)
			}
		}
	}
	// Add any chains not in chainOrder (future-proofing)
	for name := range present {
		alreadyAdded := false
		for _, n := range chainNames {
			if n == name {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			chainNames = append(chainNames, name)
			m.chainSlugs[name] = strings.ToLower(name)
		}
	}

	m.tabs = append([]string{"All"}, chainNames...)
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
}

// renderTabBar renders the tab bar as a sliding marquee — the active tab is
// pinned to the left with the next tabs visible to its right. Arrows indicate
// more tabs off-screen.
func (m ProxyViewModel) renderTabBar() string {
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ui.ColorBoba).
		Bold(true).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(ui.ColorDim).
		Padding(0, 2)

	arrowStyle := lipgloss.NewStyle().Foreground(ui.ColorBoba).Bold(true)

	// Tab widths (label + 4 padding chars)
	tabWidths := make([]int, len(m.tabs))
	totalWidth := 0
	for i, tab := range m.tabs {
		tabWidths[i] = len(tab) + 4
		totalWidth += tabWidths[i]
	}

	availWidth := m.width - 4
	if availWidth < 20 {
		availWidth = 20
	}

	// If everything fits, render all tabs normally
	if totalWidth <= availWidth {
		var tabs []string
		for i, tab := range m.tabs {
			if i == m.activeTab {
				tabs = append(tabs, activeStyle.Render(tab))
			} else {
				tabs = append(tabs, inactiveStyle.Render(tab))
			}
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

		sepLen := m.width - 4
		if sepLen > 80 {
			sepLen = 80
		}
		border := lipgloss.NewStyle().Foreground(ui.ColorDim).Render("  " + strings.Repeat("━", sepLen))
		return "  " + row + "\n" + border
	}

	// Marquee: active tab pinned left, fill remaining width with tabs to the right
	hasLeft := m.activeTab > 0
	leftArrow := "◀ "
	rightArrow := " ▶"
	arrowW := 3

	// Reserve space for arrows
	windowW := availWidth
	if hasLeft {
		windowW -= arrowW
	}
	windowW -= arrowW // always reserve right arrow space

	// Fill from active tab rightward
	var visible []int
	usedWidth := 0
	for i := m.activeTab; i < len(m.tabs); i++ {
		if usedWidth+tabWidths[i] > windowW && len(visible) > 0 {
			break
		}
		visible = append(visible, i)
		usedWidth += tabWidths[i]
	}

	hasRight := visible[len(visible)-1] < len(m.tabs)-1

	// Render
	var row string
	if hasLeft {
		row += arrowStyle.Render(leftArrow)
	}

	var parts []string
	for _, idx := range visible {
		if idx == m.activeTab {
			parts = append(parts, activeStyle.Render(m.tabs[idx]))
		} else {
			parts = append(parts, inactiveStyle.Render(m.tabs[idx]))
		}
	}
	row += lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	if hasRight {
		row += arrowStyle.Render(rightArrow)
	}

	sepLen := m.width - 4
	if sepLen > 80 {
		sepLen = 80
	}
	border := lipgloss.NewStyle().Foreground(ui.ColorDim).Render("  " + strings.Repeat("━", sepLen))

	return "  " + row + "\n" + border
}

// renderViewportContent returns the activity log for the scrollable viewport.
func (m ProxyViewModel) renderViewportContent() string {
	return m.renderLog()
}

// renderChainPortfolio renders the portfolio panel for a specific chain,
// using data from the chain-specific API call (chainPortfolio).
func (m ProxyViewModel) renderChainPortfolio(chainName string) string {
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
	symStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true)

	// Loading state
	if m.chainPortfolio == nil || m.chainPortfolioLoading {
		loadingMsg := dimStyle.Italic(true).
			Render("  " + m.spinner.View() + " Loading " + chainName + " portfolio...")
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorGold).
			Padding(0, 2).
			Render(loadingMsg)
	}

	// Error state
	if m.chainPortfolio.Error != "" {
		errMsg := dimStyle.Italic(true).Render("  " + chainName + " portfolio unavailable")
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorDim).
			Padding(0, 2).
			Render(errMsg)
	}

	p := m.chainPortfolio
	var lines []string

	// Header: chain name + total value
	headerLine := fmt.Sprintf("  %s  Total: %s",
		titleStyle.Render(strings.ToUpper(chainName)),
		formatter.FormatUSD(p.TotalValueUSD))
	lines = append(lines, headerLine)
	lines = append(lines, "")

	// Native balances
	if len(p.NativeBalances) > 0 {
		maxSymLen := 0
		for _, nb := range p.NativeBalances {
			if len(nb.Symbol) > maxSymLen {
				maxSymLen = len(nb.Symbol)
			}
		}
		for _, nb := range p.NativeBalances {
			dot := lipgloss.NewStyle().Foreground(ui.ColorCyan).Render("●")
			goldStyle := lipgloss.NewStyle().Foreground(ui.ColorGold)
			paddedSym := nb.Symbol + strings.Repeat(" ", maxSymLen-len(nb.Symbol))
			balStr := fmt.Sprintf("%.3f", nb.Balance)
			usdStr := goldStyle.Render(fmt.Sprintf("$%.2f", nb.BalanceUSD))
			lines = append(lines, fmt.Sprintf("  %s %s  %s  %s",
				dot,
				symStyle.Render(paddedSym),
				balStr,
				usdStr))
		}
		lines = append(lines, "")
	}

	// Positions — all of them, server already filtered by chain_id
	if len(p.Positions) == 0 {
		lines = append(lines, dimStyle.Render("  No positions on "+chainName))
	} else {
		goldStyle := lipgloss.NewStyle().Foreground(ui.ColorGold)

		// Find max symbol length for padding
		maxPosSymLen := 0
		for _, pos := range p.Positions {
			if len(pos.Symbol) > maxPosSymLen {
				maxPosSymLen = len(pos.Symbol)
			}
		}

		var posTotal float64
		for _, pos := range p.Positions {
			posTotal += pos.ValueUSD
		}

		// Pre-format all values to find max widths for alignment
		type posRow struct {
			symbol   string
			valStr   string
			allocStr string
			pnlStr   string
		}
		var rows []posRow
		maxValLen := 0
		maxAllocLen := 0
		for _, pos := range p.Positions {
			alloc := 0.0
			if posTotal > 0 {
				alloc = (pos.ValueUSD / posTotal) * 100
			}
			valStr := fmt.Sprintf("$%.2f", pos.ValueUSD)
			allocStr := fmt.Sprintf("%.0f%%", alloc)
			pnlStr := formatter.FormatPercent(pos.PnlPercent)
			if len(valStr) > maxValLen {
				maxValLen = len(valStr)
			}
			if len(allocStr) > maxAllocLen {
				maxAllocLen = len(allocStr)
			}
			rows = append(rows, posRow{
				symbol:   pos.Symbol,
				valStr:   valStr,
				allocStr: allocStr,
				pnlStr:   pnlStr,
			})
		}

		for _, r := range rows {
			paddedSym := r.symbol + strings.Repeat(" ", maxPosSymLen-len(r.symbol))
			paddedVal := strings.Repeat(" ", maxValLen-len(r.valStr)) + r.valStr
			paddedAlloc := strings.Repeat(" ", maxAllocLen-len(r.allocStr)) + r.allocStr
			line := fmt.Sprintf("  %s  %s  %s  %s",
				symStyle.Render(paddedSym),
				goldStyle.Render(paddedVal),
				dimStyle.Render(paddedAlloc),
				r.pnlStr)
			lines = append(lines, line)
		}
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorGold).
		BorderTop(true).
		Padding(0, 2).
		Render(content)
}

// renderLog renders the activity log entries for the viewport.
func (m ProxyViewModel) renderLog() string {
	if len(m.logEntries) == 0 {
		return m.renderIdleText()
	}

	var blocks []string
	for _, entry := range m.logEntries {
		block := m.formatLogEntry(entry)
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n")
}

func (m ProxyViewModel) renderConfigPanel() string {
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(8)
	valStyle := lipgloss.NewStyle().Foreground(ui.ColorBright)

	var lines []string
	lines = append(lines, fmt.Sprintf("  %s %s",
		labelStyle.Render("Proxy"),
		valStyle.Render(fmt.Sprintf("http://127.0.0.1:%d", m.port))))
	if m.agentName != "" {
		lines = append(lines, fmt.Sprintf("  %s %s",
			labelStyle.Render("Agent"),
			valStyle.Render(m.agentName)))
	}
	if m.evmAddr != "" {
		lines = append(lines, fmt.Sprintf("  %s %s",
			labelStyle.Render("EVM"),
			valStyle.Render(truncate(m.evmAddr))))
	}
	if m.solAddr != "" {
		lines = append(lines, fmt.Sprintf("  %s %s",
			labelStyle.Render("Solana"),
			valStyle.Render(truncate(m.solAddr))))
	}

	content := strings.Join(lines, "\n")
	closeLine := dimStyle.Render("  press c to close")

	return content + "\n" + closeLine
}

// configPanelHeight returns the number of terminal lines the config panel uses.
func (m ProxyViewModel) configPanelHeight() int {
	lines := 1 // proxy line (always shown)
	if m.agentName != "" {
		lines++
	}
	if m.evmAddr != "" {
		lines++
	}
	if m.solAddr != "" {
		lines++
	}
	lines++ // "press c to close" line
	return lines
}

func (m ProxyViewModel) renderSpecLine() string {
	dim := lipgloss.NewStyle().Foreground(ui.ColorDim)
	val := lipgloss.NewStyle().Foreground(ui.ColorBright)
	sep := dim.Render(" · ")

	var parts []string

	if m.agentName != "" {
		parts = append(parts, dim.Render("agent ")+val.Render(m.agentName))
	}

	parts = append(parts, dim.Render("proxy ")+val.Render(fmt.Sprintf(":%d", m.port)))

	if m.solAddr != "" {
		parts = append(parts, dim.Render("sol ")+val.Render(truncate(m.solAddr)))
	}
	if m.evmAddr != "" {
		parts = append(parts, dim.Render("evm ")+val.Render(truncate(m.evmAddr)))
	}

	return "  " + strings.Join(parts, sep)
}

func (m ProxyViewModel) renderStatsBar() string {
	// Pulsing alive indicator -- alternates between bright and dim each second
	var aliveDot string
	if m.idleFrame%2 == 0 {
		aliveDot = lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("●")
	} else {
		aliveDot = lipgloss.NewStyle().Foreground(lipgloss.Color("#2D8B46")).Render("●")
	}

	uptime := time.Since(m.startTime)
	uptimeStr := formatUptime(uptime)

	reqStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	uptimeStyle := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	errStyle := lipgloss.NewStyle().Foreground(ui.ColorRed)
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)

	parts := []string{
		fmt.Sprintf("  %s %s", aliveDot, reqStyle.Render(fmt.Sprintf("%d requests", m.requestCount))),
		fmt.Sprintf("%s %s", dimStyle.Render("^"), uptimeStyle.Render(uptimeStr)),
	}

	if m.errorCount > 0 {
		parts = append(parts, fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(ui.ColorRed).Render("!"),
			errStyle.Render(fmt.Sprintf("%d errors", m.errorCount))))
	} else {
		parts = append(parts, fmt.Sprintf("%s %s",
			dimStyle.Render("~"),
			lipgloss.NewStyle().Foreground(ui.ColorGreen).Render("0 errors")))
	}

	return strings.Join(parts, "  ")
}

func (m ProxyViewModel) renderPortfolioPanel() string {
	p := m.portfolio

	// Loading state — fetch in-flight, no data yet
	if p == nil {
		loadingMsg := lipgloss.NewStyle().Foreground(ui.ColorDim).Italic(true).
			Render("  " + m.spinner.View() + " Loading portfolio...")
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorGold).
			Padding(0, 2).
			Render(loadingMsg)
	}

	// Error state
	if p.Error != "" {
		dimMsg := lipgloss.NewStyle().Foreground(ui.ColorDim).Italic(true).
			Render("  Portfolio unavailable")
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorDim).
			Padding(0, 2).
			Render(dimMsg)
	}

	var lines []string

	// Header line: "PORTFOLIO  Total: $2,150.50    ↻ 25s"
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorGold).Bold(true)
	totalStr := formatter.FormatUSD(p.TotalValueUSD)

	// Refresh indicator: spinner when loading, pulsing dot otherwise, flash green on fresh data
	var refreshBadge string
	if m.portfolioLoading {
		refreshBadge = m.spinner.View()
	} else if m.portfolioFlash > 0 {
		refreshBadge = lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("●")
	} else {
		// Pulsing dim dot
		if m.idleFrame%2 == 0 {
			refreshBadge = lipgloss.NewStyle().Foreground(ui.ColorDim).Render("●")
		} else {
			refreshBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Render("●")
		}
	}

	headerLine := fmt.Sprintf("  %s  Total: %s  %s", titleStyle.Render("PORTFOLIO"), totalStr, refreshBadge)
	lines = append(lines, headerLine)
	lines = append(lines, "")

	// Native balances
	if len(p.NativeBalances) > 0 {
		// Find max symbol length for alignment
		maxSymLen := 0
		for _, nb := range p.NativeBalances {
			if len(nb.Symbol) > maxSymLen {
				maxSymLen = len(nb.Symbol)
			}
		}

		for _, nb := range p.NativeBalances {
			dot := lipgloss.NewStyle().Foreground(ui.ColorCyan).Render("●")
			symStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
			chainStyle := lipgloss.NewStyle().Foreground(ui.ColorDim)
			goldStyle := lipgloss.NewStyle().Foreground(ui.ColorGold)
			// Pad symbol to max length for alignment
			paddedSym := nb.Symbol + strings.Repeat(" ", maxSymLen-len(nb.Symbol))
			balStr := fmt.Sprintf("%.3f", nb.Balance)
			usdStr := goldStyle.Render(fmt.Sprintf("$%.2f", nb.BalanceUSD))
			chain := ""
			if nb.ChainName != "" {
				chain = chainStyle.Render("  (" + nb.ChainName + ")")
			}
			line := fmt.Sprintf("  %s %s  %s  %s%s",
				dot,
				symStyle.Render(paddedSym),
				balStr,
				usdStr,
				chain)
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}

	// Positions (max 4)
	if len(p.Positions) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorDim).Render("  No positions"))
	} else {
		shown := p.Positions
		if len(shown) > 4 {
			shown = shown[:4]
		}
		for _, pos := range shown {
			symStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
			valStr := formatter.FormatUSD(pos.ValueUSD)
			pnlStr := formatter.FormatPercent(pos.PnlPercent)
			line := fmt.Sprintf("  %s  %s  %s",
				symStyle.Render(pos.Symbol),
				valStr,
				pnlStr)
			lines = append(lines, line)
		}
		if len(p.Positions) > 4 {
			more := len(p.Positions) - 4
			lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorDim).
				Render(fmt.Sprintf("  +%d more", more)))
		}
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorGold).
		BorderTop(true).
		Padding(0, 2).
		Render(content)
}

func formatUptime(d time.Duration) string {
	totalSec := int(d.Seconds())
	h := totalSec / 3600
	min := (totalSec % 3600) / 60
	sec := totalSec % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, min, sec)
	}
	if min > 0 {
		return fmt.Sprintf("%dm %ds", min, sec)
	}
	return fmt.Sprintf("%ds", sec)
}

var idlePatterns = []string{
	"  .       Waiting for requests",
	"  ..      Waiting for requests",
	"  ...     Waiting for requests",
	"  ....    Waiting for requests",
	"  ...     Waiting for requests",
	"  ..      Waiting for requests",
}

func (m ProxyViewModel) renderIdleText() string {
	frame := m.idleFrame % len(idlePatterns)
	return lipgloss.NewStyle().Foreground(ui.ColorDim).Render("\n" + idlePatterns[frame] + "\n")
}

func (m ProxyViewModel) formatLogEntry(entry proxy.LogEntry) string {
	// Timestamp — cyan for terminal-hacker aesthetic
	ts := entry.Timestamp.Format("15:04:05")
	tsStyle := lipgloss.NewStyle().Foreground(ui.ColorCyan)

	// Category tag
	tag := getToolTag(entry.Tool)
	tagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(tag.color).
		Bold(true).
		Padding(0, 1)
	tagRendered := tagStyle.Render(tag.label)

	// Tool name
	toolColor := ui.ToolColor(entry.Tool)
	toolStyle := lipgloss.NewStyle().Foreground(toolColor).Bold(true)

	var statusIcon string
	var detail string

	switch entry.Status {
	case "pending":
		statusIcon = m.spinner.View()
		desc := toolDescriptions[entry.Tool]
		if desc == "" {
			desc = fmt.Sprintf("Calling %s...", entry.Tool)
		}
		detail = lipgloss.NewStyle().Foreground(ui.ColorDim).Italic(true).Render(desc)

	case "success":
		statusIcon = lipgloss.NewStyle().Foreground(ui.ColorGreen).Bold(true).Render("OK")
		durBadge := renderDurationBadge(entry.Duration)
		detail = durBadge
		if entry.Preview != "" {
			previewStyle := lipgloss.NewStyle().Foreground(ui.ColorBright)
			detail += " " + lipgloss.NewStyle().Foreground(ui.ColorDim).Render("->") + " " + previewStyle.Render(entry.Preview)
		}

	case "error":
		statusIcon = lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true).Render("ERR")
		durStr := formatDuration(entry.Duration)
		errMsg := entry.Error
		if len(errMsg) > 80 {
			errMsg = errMsg[:77] + "..."
		}
		detail = lipgloss.NewStyle().Foreground(ui.ColorDim).Render(durStr) +
			"  " + lipgloss.NewStyle().Foreground(ui.ColorRed).Bold(true).Render(errMsg)
	}

	statusLine := fmt.Sprintf("  %s %s %s %s %s",
		tsStyle.Render(ts),
		tagRendered,
		toolStyle.Render(entry.Tool),
		statusIcon,
		detail,
	)

	// Append full formatted output below the status line for successful calls
	if entry.Status == "success" && entry.FormattedOutput != "" {
		indented := indentBlock(entry.FormattedOutput, "    ")
		return statusLine + "\n" + indented + "\n"
	}

	return statusLine
}

// indentBlock prepends a prefix to every line of a multi-line string.
func indentBlock(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func renderDurationBadge(d time.Duration) string {
	ms := d.Milliseconds()
	durStr := formatDuration(d)
	icon := "~"

	var color lipgloss.Color
	switch {
	case ms < 500:
		color = ui.ColorGreen
		icon = ">"
	case ms < 2000:
		color = ui.ColorGold
		icon = "~"
	default:
		color = ui.ColorRed
		icon = "!"
	}

	badgeStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	return badgeStyle.Render(fmt.Sprintf("%s %s", icon, durStr))
}

func listenForLogs(ch <-chan proxy.LogEntry) tea.Cmd {
	return func() tea.Msg {
		entry := <-ch
		return LogMsg(entry)
	}
}

func truncate(addr string) string {
	if len(addr) >= 10 {
		return addr[:6] + "..." + addr[len(addr)-4:]
	}
	return addr
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
