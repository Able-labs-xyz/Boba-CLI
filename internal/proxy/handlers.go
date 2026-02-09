package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tradeboba/boba-cli/internal/auth"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/formatter"
	"github.com/tradeboba/boba-cli/internal/logger"
)

// noRedirectClient returns an HTTP client that refuses to follow redirects,
// preventing Authorization headers from being forwarded to unintended hosts.
func noRedirectClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirects are not followed for requests carrying credentials")
		},
	}
}

// toolDescriptions maps tool names to human-friendly status strings shown in
// the TUI while a request is in flight.
var toolDescriptions = map[string]string{
	"get_portfolio":       "Fetching portfolio...",
	"get_portfolio_pnl":   "Getting P&L data...",
	"get_swap_price":      "Getting swap quote...",
	"execute_swap":        "Executing trade...",
	"get_token_info":      "Looking up token...",
	"search_tokens":       "Searching tokens...",
	"get_trending_tokens": "Getting trending...",
	"get_brewing_status":  "Checking new launches...",
	"get_recent_launches": "Getting recent launches...",
	"stream_launches":     "Streaming launches...",
	"get_wallet_balance":  "Checking wallet...",
	"create_limit_order":  "Creating limit order...",
	"get_limit_orders":    "Getting limit orders...",
}

// callRequest is the JSON body expected by the /call endpoint.
// Accepts both MCP protocol format (name/arguments) and TS-compat format (tool/args).
type callRequest struct {
	Name      string                 `json:"name"`
	Tool      string                 `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Args      map[string]any `json:"args"`
}

// toolName returns the tool name from whichever field was provided.
func (c *callRequest) toolName() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Tool
}

// toolArgs returns the arguments from whichever field was provided.
func (c *callRequest) toolArgs() map[string]any {
	if c.Arguments != nil {
		return c.Arguments
	}
	return c.Args
}

// handleHealth returns basic server health information. This endpoint does not
// require authentication so that monitoring tools can reach it.
func (s *ProxyServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	agentName := ""
	agentID := ""

	tokens, err := config.GetTokens()
	if err == nil && tokens != nil {
		agentName = tokens.AgentName
		agentID = tokens.AgentID
	} else {
		c := config.Load()
		if c.Tokens != nil {
			agentName = c.Tokens.AgentName
			agentID = c.Tokens.AgentID
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"agent":    agentName,
		"agentId":  agentID,
		"requests": s.getRequestCount(),
	})
}

// handleTools proxies the tool-list request to the MCP backend and returns the
// response as-is. The agent's wallet addresses and sub-org are forwarded as
// headers so the backend can filter the tool set.
func (s *ProxyServer) handleTools(w http.ResponseWriter, r *http.Request) {
	tokens, err := auth.EnsureAuthenticated()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("authentication failed: %v", err)})
		return
	}

	client := noRedirectClient(30 * time.Second)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tools", config.GetMCPURL()), nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to create request: %v", err)})
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	req.Header.Set("X-Agent-EVM-Address", tokens.EVMAddress)
	req.Header.Set("X-Agent-Solana-Address", tokens.SolanaAddress)
	req.Header.Set("X-Agent-Sub-Org-Id", tokens.SubOrganizationID)

	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("upstream request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Forward the response headers and body as-is.
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleCall proxies a tool invocation to the MCP backend. It auto-fills
// parameters, logs the request lifecycle, and retries once on auth errors.
func (s *ProxyServer) handleCall(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1 MB to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req callRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid request body: %v", err)})
		return
	}

	// Normalize: merge tool/args into name/arguments
	toolName := req.toolName()
	args := req.toolArgs()
	if args == nil {
		args = make(map[string]any)
	}

	// Determine a friendly description for the log entry.
	desc := toolDescriptions[toolName]
	if desc == "" {
		desc = fmt.Sprintf("Calling %s...", toolName)
	}

	// Log a pending entry so the TUI can show progress immediately.
	s.sendLog(LogEntry{
		Tool:    toolName,
		Status:  "pending",
		Preview: desc,
	})

	start := time.Now()

	// Authenticate and auto-fill parameters.
	tokens, err := auth.EnsureAuthenticated()
	if err != nil {
		duration := time.Since(start)
		errMsg := fmt.Sprintf("authentication failed: %v", err)
		s.sendLog(LogEntry{
			Tool:     toolName,
			Status:   "error",
			Duration: duration,
			Error:    errMsg,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		return
	}

	AutoFillParams(toolName, args, tokens)

	// Forward the call to the MCP backend.
	respBody, statusCode, err := s.doMCPCall(toolName, args, tokens)
	if err != nil {
		duration := time.Since(start)
		errMsg := fmt.Sprintf("upstream request failed: %v", err)
		s.sendLog(LogEntry{
			Tool:     toolName,
			Status:   "error",
			Duration: duration,
			Error:    errMsg,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		return
	}

	// Retry once on auth errors (401 / 403).
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		logger.Debug("received auth error from upstream, re-authenticating", "status", statusCode)
		newTokens, authErr := auth.Authenticate()
		if authErr == nil {
			tokens = newTokens
			AutoFillParams(toolName, args, tokens)
			respBody, statusCode, err = s.doMCPCall(toolName, args, tokens)
			if err != nil {
				duration := time.Since(start)
				errMsg := fmt.Sprintf("upstream request failed after retry: %v", err)
				s.sendLog(LogEntry{
					Tool:     toolName,
					Status:   "error",
					Duration: duration,
					Error:    errMsg,
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
				return
			}
		}
	}

	duration := time.Since(start)
	s.incrementRequests()

	// Parse the response for logging.
	var responseData any
	_ = json.Unmarshal(respBody, &responseData)

	preview := formatter.FormatToolPreview(toolName, responseData)
	formatted := formatter.FormatToolResult(toolName, responseData)

	if statusCode >= 200 && statusCode < 300 {
		s.sendLog(LogEntry{
			Tool:            toolName,
			Status:          "success",
			Duration:        duration,
			Preview:         preview,
			FormattedOutput: formatted,
		})
	} else {
		s.sendLog(LogEntry{
			Tool:     toolName,
			Status:   "error",
			Duration: duration,
			Error:    string(respBody),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(respBody)
}

// doMCPCall sends the tool call request to the MCP backend and returns the raw
// response body, HTTP status code, and any transport error.
// Uses "tool"/"args" field names matching the TS proxy format that the MCP backend expects.
func (s *ProxyServer) doMCPCall(tool string, args map[string]any, tokens *config.AuthTokens) ([]byte, int, error) {
	// Send as { "tool": ..., "args": ... } to match what the MCP backend expects
	payload := map[string]any{
		"tool": tool,
		"args": args,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := noRedirectClient(60 * time.Second)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/call", config.GetMCPURL()), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	httpReq.Header.Set("X-Agent-EVM-Address", tokens.EVMAddress)
	httpReq.Header.Set("X-Agent-Solana-Address", tokens.SolanaAddress)
	httpReq.Header.Set("X-Agent-Sub-Org-Id", tokens.SubOrganizationID)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// handleStream proxies a Server-Sent Events stream from the MCP backend to the
// client, flushing each chunk as it arrives.
func (s *ProxyServer) handleStream(w http.ResponseWriter, r *http.Request) {
	tokens, err := auth.EnsureAuthenticated()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("authentication failed: %v", err)})
		return
	}

	client := &http.Client{
		// No timeout — SSE streams are long-lived.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirects are not followed for requests carrying credentials")
		},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/stream", config.GetMCPURL()), nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to create request: %v", err)})
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	req.Header.Set("X-Agent-EVM-Address", tokens.EVMAddress)
	req.Header.Set("X-Agent-Solana-Address", tokens.SolanaAddress)
	req.Header.Set("X-Agent-Sub-Org-Id", tokens.SubOrganizationID)

	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("upstream request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback: copy the entire body at once if flushing is not supported.
		io.Copy(w, resp.Body)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				logger.Debug("stream write error", "error", writeErr)
				return
			}
			flusher.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				logger.Debug("stream read error", "error", readErr)
			}
			return
		}
	}
}
