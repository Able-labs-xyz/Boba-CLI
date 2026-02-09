package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Run as MCP server (JSON-RPC stdio mode)",
	Hidden: true,
	RunE:   runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	if !config.HasCredentials() {
		return fmt.Errorf("no credentials. Run 'boba login' first")
	}

	port := config.GetProxyPort()
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(proxyURL + "/health")
	if err != nil {
		return fmt.Errorf("proxy not running. Start it with 'boba start' first")
	}
	resp.Body.Close()

	sessionToken, err := config.GetSessionToken()
	if err != nil || sessionToken == "" {
		return fmt.Errorf("proxy session token not found. Is the proxy running?")
	}

	bridge := mcp.NewBridge(proxyURL, sessionToken)
	return bridge.Run()
}
