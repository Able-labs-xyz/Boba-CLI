package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/proxy"
	"github.com/tradeboba/boba-cli/internal/tui"
	"github.com/tradeboba/boba-cli/internal/ui"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the Boba proxy",
	RunE:  runStart,
}

var flagPort int

func init() {
	startCmd.Flags().IntVarP(&flagPort, "port", "p", 0, "Port to run proxy on")
}

func runStart(cmd *cobra.Command, args []string) error {
	if !config.HasCredentials() {
		return fmt.Errorf("no credentials configured. Run 'boba login' first")
	}

	port := flagPort
	if port == 0 {
		port = config.GetProxyPort()
	}

	server, err := proxy.NewProxyServer(port)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	agentName := ""
	evmAddr := ""
	solAddr := ""
	tokens, err := config.GetTokens()
	if err == nil {
		agentName = tokens.AgentName
		evmAddr = tokens.EVMAddress
		solAddr = tokens.SolanaAddress
	}

	model := tui.NewProxyViewModel(server, agentName, evmAddr, solAddr, port)
	p := tea.NewProgram(model, tea.WithAltScreen())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		p.Send(tea.Quit())
	}()

	if _, err := p.Run(); err != nil {
		_ = server.Stop()
		return fmt.Errorf("TUI error: %w", err)
	}

	_ = server.Stop()
	fmt.Println(ui.DimStyle.Render("\n  Proxy stopped. Goodbye!\n"))
	return nil
}
