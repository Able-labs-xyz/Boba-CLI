package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tradeboba/boba-cli/internal/auth"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/logger"
)

// LogEntry represents a single proxy request log item displayed in the TUI.
type LogEntry struct {
	Tool            string
	Status          string // "pending", "success", "error"
	Duration        time.Duration
	Preview         string // Short one-line summary for the status line
	FormattedOutput string // Full multi-line rich formatted output (charts, tables, boxes)
	Timestamp       time.Time
	Error           string
}

// ProxyServer is an HTTP proxy that sits between AI agents and the Boba MCP
// backend. It handles authentication, parameter auto-fill, and request logging.
type ProxyServer struct {
	server       *http.Server
	port         int
	sessionToken string
	logChan      chan LogEntry
	requestCount int64
	mu           sync.RWMutex
}

// NewProxyServer creates a new proxy server bound to 127.0.0.1 on the given
// port. A cryptographically random session token is generated and stored in the
// system keyring so that only authorised callers can reach the proxy.
func NewProxyServer(port int) (*ProxyServer, error) {
	// Verify the MCP URL uses HTTPS or localhost to prevent credential leakage.
	mcpURL := config.GetMCPURL()
	if !config.IsHTTPSOrLocal(mcpURL) {
		return nil, fmt.Errorf("MCP URL must use HTTPS or localhost: %s", mcpURL)
	}

	// Generate a 32-byte random session token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Persist the token in the system keyring so other processes can retrieve it.
	if err := config.SetSessionToken(sessionToken); err != nil {
		return nil, fmt.Errorf("failed to store session token: %w", err)
	}

	s := &ProxyServer{
		port:         port,
		sessionToken: sessionToken,
		logChan:      make(chan LogEntry, 100),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /tools", s.withAuth(s.handleTools))
	mux.HandleFunc("POST /call", s.withAuth(s.handleCall))
	mux.HandleFunc("GET /stream", s.withAuth(s.handleStream))

	s.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s, nil
}

// Start begins listening for connections in a background goroutine. It returns
// an error if the listener cannot be created.
func (s *ProxyServer) Start() error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.server.Addr, err)
	}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("proxy server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the proxy server with a 5-second deadline and
// clears the session token from the system keyring.
func (s *ProxyServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.server.Shutdown(ctx)

	// Always attempt to clear the session token, even if shutdown had an error.
	_ = config.ClearSessionToken()

	return err
}

// LogChannel returns a read-only channel that receives log entries for every
// proxied request.
func (s *ProxyServer) LogChannel() <-chan LogEntry {
	return s.logChan
}

// SessionToken returns the session token required to authenticate with this
// proxy instance.
func (s *ProxyServer) SessionToken() string {
	return s.sessionToken
}

// Port returns the port the proxy server is bound to.
func (s *ProxyServer) Port() int {
	return s.port
}

// sendLog sends a log entry to the log channel without blocking. If the
// channel buffer is full the entry is silently dropped to avoid back-pressure
// on request processing.
func (s *ProxyServer) sendLog(entry LogEntry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	select {
	case s.logChan <- entry:
	default:
		// Channel full — drop the entry to avoid blocking the handler.
	}
}

// incrementRequests atomically increments and returns the new request count.
func (s *ProxyServer) incrementRequests() int64 {
	return atomic.AddInt64(&s.requestCount, 1)
}

// getRequestCount returns the current request count.
func (s *ProxyServer) getRequestCount() int64 {
	return atomic.LoadInt64(&s.requestCount)
}

// CallTool makes an MCP tool call directly, bypassing the HTTP layer. This is
// used by the TUI for background polling (e.g. portfolio updates) without going
// through the HTTP loopback. It handles authentication, parameter auto-fill,
// and retries once on 401/403 — the same logic as handleCall.
func (s *ProxyServer) CallTool(tool string, args map[string]any) ([]byte, error) {
	tokens, err := auth.EnsureAuthenticated()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	AutoFillParams(tool, args, tokens)

	respBody, statusCode, err := s.doMCPCall(tool, args, tokens)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}

	// Retry once on auth errors.
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		logger.Debug("CallTool: auth error from upstream, re-authenticating", "status", statusCode)
		newTokens, authErr := auth.Authenticate()
		if authErr != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", authErr)
		}
		AutoFillParams(tool, args, newTokens)
		respBody, statusCode, err = s.doMCPCall(tool, args, newTokens)
		if err != nil {
			return nil, fmt.Errorf("upstream request failed after retry: %w", err)
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("upstream returned status %d: %s", statusCode, string(respBody))
	}

	return respBody, nil
}
