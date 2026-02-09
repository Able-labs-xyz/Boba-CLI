package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	KeychainService      = "boba-cli"
	KeychainSecret       = "agent-secret"
	KeychainAccessToken  = "access-token"
	KeychainRefreshToken = "refresh-token"
	KeychainSessionToken = "session-token"

	DefaultMCPURL   = "https://mcp-skunk.up.railway.app"
	DefaultAuthURL  = "https://krakend-skunk.up.railway.app/v2"
	DefaultPort     = 3456
	DefaultLogLevel = "info"
)

// Env var fallback names for headless systems without a keyring.
var envVarMap = map[string]string{
	KeychainSecret:       "BOBA_AGENT_SECRET",
	KeychainAccessToken:  "BOBA_ACCESS_TOKEN",
	KeychainRefreshToken: "BOBA_REFRESH_TOKEN",
	KeychainSessionToken: "BOBA_SESSION_TOKEN",
}

// keyringOK is true when the OS keyring backend is usable.
var keyringOK = sync.OnceValue(func() bool {
	const probe = "boba-cli-keyring-probe"
	if err := keyring.Set(KeychainService, probe, "ok"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: system keyring unavailable, falling back to environment variables (BOBA_AGENT_SECRET, BOBA_ACCESS_TOKEN, etc.)")
		return false
	}
	_ = keyring.Delete(KeychainService, probe)
	return true
})

func secureGet(account string) (string, error) {
	if keyringOK() {
		if val, err := keyring.Get(KeychainService, account); err == nil {
			return val, nil
		}
	}
	if envVar, ok := envVarMap[account]; ok {
		if val := os.Getenv(envVar); val != "" {
			return val, nil
		}
	}
	return "", fmt.Errorf("%s not found in keyring or environment", account)
}

func secureSet(account, value string) error {
	if keyringOK() {
		return keyring.Set(KeychainService, account, value)
	}
	// No keyring available — user manages secrets via env vars.
	return nil
}

func secureDelete(account string) {
	if keyringOK() {
		_ = keyring.Delete(KeychainService, account)
	}
}

var AllowedHosts = []string{
	"mcp-skunk.up.railway.app",
	"krakend-skunk.up.railway.app",
	"localhost",
	"127.0.0.1",
}

type AgentCredentials struct {
	AgentID     string `json:"agentId"`
	AgentSecret string `json:"-"`
	Name        string `json:"name,omitempty"`
}

type AuthTokens struct {
	AccessToken           string `json:"accessToken"`
	RefreshToken          string `json:"-"`
	AccessTokenExpiresAt  string `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt"`
	AgentID               string `json:"agentId"`
	AgentName             string `json:"agentName"`
	EVMAddress            string `json:"evmAddress"`
	SolanaAddress         string `json:"solanaAddress"`
	SubOrganizationID     string `json:"subOrganizationId"`
}

type BobaConfig struct {
	MCPURL      string `json:"mcpUrl"`
	AuthURL     string `json:"authUrl"`
	ProxyPort   int    `json:"proxyPort"`
	LogLevel    string `json:"logLevel"`
	Credentials *struct {
		AgentID string `json:"agentId"`
		Name    string `json:"name,omitempty"`
	} `json:"credentials,omitempty"`
	Tokens *struct {
		AccessTokenExpiresAt  string `json:"accessTokenExpiresAt"`
		RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt"`
		AgentID               string `json:"agentId"`
		AgentName             string `json:"agentName"`
		EVMAddress            string `json:"evmAddress"`
		SolanaAddress         string `json:"solanaAddress"`
		SubOrganizationID     string `json:"subOrganizationId"`
	} `json:"tokens,omitempty"`
}

var cfg *BobaConfig
var configPath string

func init() {
	configPath = getConfigPath()
}

func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "boba-cli", "config.json")
}

func ConfigPath() string {
	return configPath
}

func Load() *BobaConfig {
	if cfg != nil {
		return cfg
	}

	cfg = &BobaConfig{
		MCPURL:    DefaultMCPURL,
		AuthURL:   DefaultAuthURL,
		ProxyPort: DefaultPort,
		LogLevel:  DefaultLogLevel,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		migrateFromTS()
		return cfg
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg
	}

	if cfg.MCPURL == "" {
		cfg.MCPURL = DefaultMCPURL
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = DefaultAuthURL
	}
	if cfg.ProxyPort == 0 {
		cfg.ProxyPort = DefaultPort
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultLogLevel
	}

	return cfg
}

func save() error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// Credentials

func HasCredentials() bool {
	c := Load()
	if c.Credentials == nil || c.Credentials.AgentID == "" {
		return false
	}
	secret, err := secureGet(KeychainSecret)
	return err == nil && secret != ""
}

func GetCredentials() (*AgentCredentials, error) {
	c := Load()
	if c.Credentials == nil || c.Credentials.AgentID == "" {
		return nil, fmt.Errorf("no credentials configured")
	}

	secret, err := secureGet(KeychainSecret)
	if err != nil {
		return nil, fmt.Errorf("agent secret not found in keyring or BOBA_AGENT_SECRET env")
	}

	return &AgentCredentials{
		AgentID:     c.Credentials.AgentID,
		AgentSecret: secret,
		Name:        c.Credentials.Name,
	}, nil
}

func SetCredentials(agentID, secret, name string) error {
	c := Load()
	c.Credentials = &struct {
		AgentID string `json:"agentId"`
		Name    string `json:"name,omitempty"`
	}{
		AgentID: agentID,
		Name:    name,
	}

	if err := secureSet(KeychainSecret, secret); err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	return save()
}

func ClearCredentials() error {
	c := Load()
	c.Credentials = nil
	c.Tokens = nil

	secureDelete(KeychainSecret)
	secureDelete(KeychainAccessToken)
	secureDelete(KeychainRefreshToken)
	secureDelete(KeychainSessionToken)

	return save()
}

// Tokens

func GetTokens() (*AuthTokens, error) {
	c := Load()
	if c.Tokens == nil {
		return nil, fmt.Errorf("no auth tokens")
	}

	accessToken, err := secureGet(KeychainAccessToken)
	if err != nil {
		return nil, fmt.Errorf("access token not found in keyring or BOBA_ACCESS_TOKEN env")
	}

	refreshToken, _ := secureGet(KeychainRefreshToken)

	return &AuthTokens{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  c.Tokens.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: c.Tokens.RefreshTokenExpiresAt,
		AgentID:               c.Tokens.AgentID,
		AgentName:             c.Tokens.AgentName,
		EVMAddress:            c.Tokens.EVMAddress,
		SolanaAddress:         c.Tokens.SolanaAddress,
		SubOrganizationID:     c.Tokens.SubOrganizationID,
	}, nil
}

func SetTokens(tokens *AuthTokens) error {
	c := Load()
	c.Tokens = &struct {
		AccessTokenExpiresAt  string `json:"accessTokenExpiresAt"`
		RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt"`
		AgentID               string `json:"agentId"`
		AgentName             string `json:"agentName"`
		EVMAddress            string `json:"evmAddress"`
		SolanaAddress         string `json:"solanaAddress"`
		SubOrganizationID     string `json:"subOrganizationId"`
	}{
		AccessTokenExpiresAt:  tokens.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: tokens.RefreshTokenExpiresAt,
		AgentID:               tokens.AgentID,
		AgentName:             tokens.AgentName,
		EVMAddress:            tokens.EVMAddress,
		SolanaAddress:         tokens.SolanaAddress,
		SubOrganizationID:     tokens.SubOrganizationID,
	}

	if err := secureSet(KeychainAccessToken, tokens.AccessToken); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	if tokens.RefreshToken != "" {
		if err := secureSet(KeychainRefreshToken, tokens.RefreshToken); err != nil {
			return fmt.Errorf("failed to store refresh token: %w", err)
		}
	}

	return save()
}

func IsTokenExpired() bool {
	c := Load()
	if c.Tokens == nil || c.Tokens.AccessTokenExpiresAt == "" {
		return true
	}

	expiresAt, err := parseTime(c.Tokens.AccessTokenExpiresAt)
	if err != nil {
		return true
	}

	// Consider expired 1 minute before actual expiry (matches TS version)
	return time.Now().After(expiresAt.Add(-60 * time.Second))
}

// parseTime tries multiple common timestamp formats to handle whatever the
// backend returns (with or without fractional seconds, Z or offset).
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,                // 2006-01-02T15:04:05.999999999Z07:00
		time.RFC3339,                    // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05.000Z0700", // milliseconds without colon
		"2006-01-02T15:04:05Z0700",     // no colon in offset
		"2006-01-02 15:04:05",          // plain datetime
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

// Session Token

func GetSessionToken() (string, error) {
	return secureGet(KeychainSessionToken)
}

func SetSessionToken(token string) error {
	return secureSet(KeychainSessionToken, token)
}

func ClearSessionToken() error {
	secureDelete(KeychainSessionToken)
	return nil
}

// Config Getters/Setters

func GetMCPURL() string {
	return Load().MCPURL
}

func SetMCPURL(urlStr string, force bool) error {
	if !force && !IsAllowedURL(urlStr) {
		return fmt.Errorf("blocked: %s is not an allowed host. Allowed: %v. Use --force to override", urlStr, AllowedHosts)
	}
	c := Load()
	c.MCPURL = urlStr
	return save()
}

func GetAuthURL() string {
	return Load().AuthURL
}

func SetAuthURL(urlStr string, force bool) error {
	if !force && !IsAllowedURL(urlStr) {
		return fmt.Errorf("blocked: %s is not an allowed host. Allowed: %v. Use --force to override", urlStr, AllowedHosts)
	}
	c := Load()
	c.AuthURL = urlStr
	return save()
}

func GetProxyPort() int {
	return Load().ProxyPort
}

func SetProxyPort(port int) error {
	c := Load()
	c.ProxyPort = port
	return save()
}

func GetLogLevel() string {
	return Load().LogLevel
}

func Reset() error {
	cfg = &BobaConfig{
		MCPURL:    DefaultMCPURL,
		AuthURL:   DefaultAuthURL,
		ProxyPort: DefaultPort,
		LogLevel:  DefaultLogLevel,
	}

	secureDelete(KeychainSecret)
	secureDelete(KeychainAccessToken)
	secureDelete(KeychainRefreshToken)
	secureDelete(KeychainSessionToken)

	return save()
}

// URL Allowlist

func IsAllowedURL(urlStr string) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	hostname := parsed.Hostname()
	for _, h := range AllowedHosts {
		if h == hostname {
			return true
		}
	}
	return false
}

func IsHTTPSOrLocal(urlStr string) bool {
	if strings.HasPrefix(urlStr, "https://") {
		return true
	}
	if strings.HasPrefix(urlStr, "http://localhost") || strings.HasPrefix(urlStr, "http://127.0.0.1") {
		return true
	}
	return false
}

// Migration from TS version

func migrateFromTS() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	tsPaths := []string{
		filepath.Join(home, ".config", "boba-cli", "config.json"),
		filepath.Join(home, "Library", "Preferences", "boba-cli-nodejs", "config.json"),
	}

	for _, p := range tsPaths {
		if p == configPath {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		var tsConfig map[string]any
		if err := json.Unmarshal(data, &tsConfig); err != nil {
			continue
		}

		if v, ok := tsConfig["mcpUrl"].(string); ok && v != "" {
			cfg.MCPURL = v
		}
		if v, ok := tsConfig["authUrl"].(string); ok && v != "" {
			cfg.AuthURL = v
		}
		if v, ok := tsConfig["proxyPort"].(float64); ok && v > 0 {
			cfg.ProxyPort = int(v)
		}
		if v, ok := tsConfig["logLevel"].(string); ok && v != "" {
			cfg.LogLevel = v
		}

		if creds, ok := tsConfig["credentials"].(map[string]any); ok {
			agentID, _ := creds["agentId"].(string)
			name, _ := creds["name"].(string)
			if agentID != "" {
				cfg.Credentials = &struct {
					AgentID string `json:"agentId"`
					Name    string `json:"name,omitempty"`
				}{
					AgentID: agentID,
					Name:    name,
				}
			}
		}

		_ = save()
		return
	}
}
