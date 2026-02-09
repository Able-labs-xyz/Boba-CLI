package proxy

import (
	"regexp"
	"strings"

	"github.com/tradeboba/boba-cli/internal/config"
)

// userIDTools is the set of tools whose user_id / userId parameter should be
// auto-filled with the authenticated agent's ID when it is missing or fake.
var userIDTools = map[string]bool{
	"get_portfolio":              true,
	"get_portfolio_summary":      true,
	"get_portfolio_pnl":          true,
	"get_trade_history":          true,
	"get_pnl_chart":              true,
	"get_user_xp":                true,
	"get_transfers":              true,
	"get_wallet_balance":         true,
	"get_limit_orders":           true,
	"get_dca_orders":             true,
	"get_twap_orders":            true,
	"get_positions":              true,
	"create_limit_order":         true,
	"cancel_limit_order":         true,
	"get_user_swaps":             true,
	"refresh_native_balances":    true,
	"start_portfolio_stream":     true,
	"get_portfolio_price_updates": true,
	"stop_portfolio_stream":      true,
}

// swapTools is the set of tools that need a from-address / taker parameter
// auto-filled with the agent's wallet address.
var swapTools = map[string]bool{
	"get_swap_price":  true,
	"get_swap_quote":  true,
	"execute_swap":    true,
	"execute_trade":   true,
}

// walletParams lists all parameter names that represent a wallet address and
// should be auto-filled when the value is a placeholder.
var walletParams = map[string]bool{
	"wallet":          true,
	"wallet_address":  true,
	"walletAddress":   true,
	"evm_address":     true,
	"taker":           true,
	"from_address":    true,
	"fromAddress":     true,
	"solana_address":  true,
}

var (
	allOnesRe  = regexp.MustCompile(`^1+$`)
	allZerosRe = regexp.MustCompile(`^0+$`)
	base58Re   = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]+$`)
)

// IsFakeID returns true when the given identifier is a placeholder that AI
// models commonly hallucinate instead of a real user / wallet ID.
func IsFakeID(id string) bool {
	if id == "" {
		return true
	}
	if allOnesRe.MatchString(id) {
		return true
	}
	if allZerosRe.MatchString(id) {
		return true
	}
	if id == "me" || id == "self" {
		return true
	}
	// EVM address (0x + 40 hex chars)
	if strings.HasPrefix(id, "0x") && len(id) == 42 {
		return true
	}
	// Solana base58 address (32-44 chars of base58)
	if len(id) >= 32 && len(id) <= 44 && base58Re.MatchString(id) {
		return true
	}
	return false
}

// IsSolanaChain returns true when the chain parameter indicates the Solana
// network. It accepts both the numeric chain ID (1399811149) and the string
// "solana" (case-insensitive).
func IsSolanaChain(chain any) bool {
	switch v := chain.(type) {
	case float64:
		return v == 1399811149
	case string:
		return strings.ToLower(v) == "solana"
	}
	return false
}

// AutoFillParams mutates args in place, replacing placeholder / missing values
// with the authenticated agent's real identifiers. This mirrors the TypeScript
// proxy's auto-fill behaviour so that AI-generated tool calls work correctly
// even when the model hallucinates IDs.
func AutoFillParams(toolName string, args map[string]any, tokens *config.AuthTokens) {
	if tokens == nil {
		return
	}

	// 1. Auto-fill user_id / userId for user-scoped tools.
	if userIDTools[toolName] {
		for _, key := range []string{"user_id", "userId"} {
			val, _ := args[key].(string)
			if val == "" || IsFakeID(val) {
				args[key] = tokens.AgentID
			}
		}
	}

	// 2. Auto-fill wallet address parameters.
	solana := IsSolanaChain(args["chain"])
	for param := range walletParams {
		val, ok := args[param].(string)
		if !ok {
			continue
		}
		if val == "my-wallet-evm" || val == "my-wallet-svm" || val == "me" || val == "self" {
			if solana || strings.Contains(param, "solana") {
				args[param] = tokens.SolanaAddress
			} else {
				args[param] = tokens.EVMAddress
			}
		}
	}

	// 3. Auto-fill swap tool from-address / taker.
	if swapTools[toolName] {
		for _, key := range []string{"from_address", "fromAddress", "taker"} {
			val, _ := args[key].(string)
			if val == "" || IsFakeID(val) {
				if solana {
					args[key] = tokens.SolanaAddress
				} else {
					args[key] = tokens.EVMAddress
				}
			}
		}
	}
}
