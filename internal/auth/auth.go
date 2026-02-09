package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tradeboba/boba-cli/internal/config"
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

type authRequest struct {
	AuthMethod  string `json:"auth_method"`
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
}

type authResponseData struct {
	SessionID             string `json:"session_id"`
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
	AgentID               string `json:"agent_id"`
	AgentName             string `json:"agent_name"`
	EVMAddress            string `json:"evm_address"`
	SolanaAddress         string `json:"solana_address"`
	SubOrganizationID     string `json:"sub_organization_id"`
}

type authResponse struct {
	Data authResponseData `json:"data"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponseData struct {
	AccessToken          string `json:"access_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
}

type refreshResponse struct {
	Data refreshResponseData `json:"data"`
}

// Authenticate performs a full authentication flow using agent credentials.
func Authenticate() (*config.AuthTokens, error) {
	creds, err := config.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	authURL := config.GetAuthURL()
	if !config.IsHTTPSOrLocal(authURL) {
		return nil, fmt.Errorf("authentication URL must use HTTPS or localhost: %s", authURL)
	}

	reqBody := authRequest{
		AuthMethod:  "agent",
		AgentID:     creds.AgentID,
		AgentSecret: creds.AgentSecret,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/user/auth/authenticate", authURL)
	client := noRedirectClient(30 * time.Second)

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	// Try parsing with { "data": { ... } } wrapper first (what TS expects)
	var authResp authResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	authData := authResp.Data

	// Fallback: if data wrapper yielded empty token, try parsing without wrapper
	if authData.AccessToken == "" {
		logger.Debug("auth response had empty access_token with data wrapper, trying direct parse")
		var directResp authResponseData
		if err := json.Unmarshal(respBody, &directResp); err == nil && directResp.AccessToken != "" {
			authData = directResp
		}
	}

	if authData.AccessToken == "" {
		logger.Debug("auth response had empty access_token after all parse attempts", "status", resp.StatusCode)
		return nil, fmt.Errorf("authentication succeeded but received empty access token — check auth response format")
	}

	tokens := &config.AuthTokens{
		AccessToken:           authData.AccessToken,
		RefreshToken:          authData.RefreshToken,
		AccessTokenExpiresAt:  authData.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: authData.RefreshTokenExpiresAt,
		AgentID:               authData.AgentID,
		AgentName:             authData.AgentName,
		EVMAddress:            authData.EVMAddress,
		SolanaAddress:         authData.SolanaAddress,
		SubOrganizationID:     authData.SubOrganizationID,
	}

	logger.Debug("authenticated successfully", "agent", tokens.AgentName, "agentId", tokens.AgentID)

	if err := config.SetTokens(tokens); err != nil {
		return nil, fmt.Errorf("failed to store tokens: %w", err)
	}

	// Register with limit orders service (non-fatal, silent)
	_ = RegisterWithLimitOrders(tokens)

	// Initialize wallet monitoring (non-fatal, silent)
	_ = InitializeWalletMonitoring(tokens)

	return tokens, nil
}

// RefreshTokens attempts to refresh the access token using the refresh token.
func RefreshTokens() (*config.AuthTokens, error) {
	existingTokens, err := config.GetTokens()
	if err != nil || existingTokens.RefreshToken == "" {
		logger.Debug("no refresh token available, falling back to full authentication")
		return Authenticate()
	}

	authURL := config.GetAuthURL()
	if !config.IsHTTPSOrLocal(authURL) {
		return nil, fmt.Errorf("authentication URL must use HTTPS or localhost: %s", authURL)
	}

	reqBody := refreshRequest{
		RefreshToken: existingTokens.RefreshToken,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/user/auth/refresh", authURL)
	client := noRedirectClient(30 * time.Second)

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var refreshResp refreshResponse
	if err := json.Unmarshal(respBody, &refreshResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	refreshData := refreshResp.Data

	// Fallback: try without data wrapper
	if refreshData.AccessToken == "" {
		var directResp refreshResponseData
		if err := json.Unmarshal(respBody, &directResp); err == nil && directResp.AccessToken != "" {
			refreshData = directResp
		}
	}

	if refreshData.AccessToken == "" {
		logger.Debug("refresh response had empty access_token after all parse attempts", "status", resp.StatusCode)
		return nil, fmt.Errorf("token refresh succeeded but received empty access token")
	}

	// Keep existing metadata, update only access token and expiry
	tokens := &config.AuthTokens{
		AccessToken:           refreshData.AccessToken,
		RefreshToken:          existingTokens.RefreshToken,
		AccessTokenExpiresAt:  refreshData.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: existingTokens.RefreshTokenExpiresAt,
		AgentID:               existingTokens.AgentID,
		AgentName:             existingTokens.AgentName,
		EVMAddress:            existingTokens.EVMAddress,
		SolanaAddress:         existingTokens.SolanaAddress,
		SubOrganizationID:     existingTokens.SubOrganizationID,
	}

	if err := config.SetTokens(tokens); err != nil {
		return nil, fmt.Errorf("failed to store refreshed tokens: %w", err)
	}

	return tokens, nil
}

// EnsureAuthenticated checks if the current tokens are valid and refreshes or
// re-authenticates as needed.
func EnsureAuthenticated() (*config.AuthTokens, error) {
	if !config.IsTokenExpired() {
		tokens, err := config.GetTokens()
		if err == nil {
			return tokens, nil
		}
	}

	// Token is expired or missing, try refresh first
	tokens, err := RefreshTokens()
	if err != nil {
		logger.Debug("token refresh failed, attempting full authentication", "error", err)
		tokens, err = Authenticate()
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	return tokens, nil
}

