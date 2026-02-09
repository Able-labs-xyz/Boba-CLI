package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/logger"
)

// RegisterWithLimitOrders registers the agent with the limit orders service.
// This is a non-fatal operation; errors are logged as warnings.
func RegisterWithLimitOrders(tokens *config.AuthTokens) error {
	authURL := config.GetAuthURL()
	baseURL := strings.Replace(authURL, "/v2", "/v2/limit", 1)
	endpoint := fmt.Sprintf("%s/agents/register", baseURL)

	body := map[string]string{
		"sub_organization_id": tokens.SubOrganizationID,
		"wallet_address":      tokens.SolanaAddress,
		"evm_address":         tokens.EVMAddress,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		logger.Warn("failed to marshal limit orders registration request", "error", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		logger.Warn("failed to create limit orders registration request", "error", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))

	client := noRedirectClient(10 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("limit orders registration request failed", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("limit orders registration returned non-OK status", "status", resp.StatusCode)
		return fmt.Errorf("limit orders registration failed with status %d", resp.StatusCode)
	}

	logger.Debug("successfully registered with limit orders service")
	return nil
}

// InitializeWalletMonitoring initializes wallet monitoring for the agent.
// This is a non-fatal operation; errors are logged as warnings.
func InitializeWalletMonitoring(tokens *config.AuthTokens) error {
	authURL := config.GetAuthURL()
	baseURL := strings.Replace(authURL, "/v2", "/v2/portfolio", 1)
	endpoint := fmt.Sprintf("%s/%s/wallets/init", baseURL, tokens.AgentID)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader([]byte("{}")))
	if err != nil {
		logger.Warn("failed to create wallet monitoring request", "error", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))

	client := noRedirectClient(10 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("wallet monitoring initialization request failed", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("wallet monitoring initialization returned non-OK status", "status", resp.StatusCode)
		return fmt.Errorf("wallet monitoring initialization failed with status %d", resp.StatusCode)
	}

	logger.Debug("successfully initialized wallet monitoring")
	return nil
}
