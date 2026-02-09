package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/tradeboba/boba-cli/internal/config"
	"github.com/tradeboba/boba-cli/internal/version"
)

type Bridge struct {
	proxyURL     string
	sessionToken string
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	client       *http.Client
}

// NewBridge creates a new MCP stdio bridge that proxies JSON-RPC requests
// to the local proxy server.
func NewBridge(proxyURL, sessionToken string) *Bridge {
	return &Bridge{
		proxyURL:     proxyURL,
		sessionToken: sessionToken,
		stdin:        os.Stdin,
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Run starts the main JSON-RPC stdio loop. It reads newline-delimited JSON-RPC
// requests from stdin, dispatches them, and writes responses to stdout.
func (b *Bridge) Run() error {
	scanner := bufio.NewScanner(b.stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			b.logError("failed to parse JSON-RPC request: %v", err)
			continue
		}

		resp := b.handleRequest(&req)
		if resp != nil {
			b.writeResponse(resp)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest dispatches a JSON-RPC request to the appropriate handler
// based on the method name.
func (b *Bridge) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return b.handleInitialize(req)
	case "notifications/initialized":
		return nil
	case "tools/list":
		return b.handleToolsList(req)
	case "tools/call":
		return b.handleToolsCall(req)
	default:
		return &JSONRPCResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize responds to the MCP initialize handshake with server
// capabilities and version information.
func (b *Bridge) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "boba",
				"version": version.Version,
			},
		},
	}
}

// handleToolsList forwards a tools/list request to the proxy and returns the
// list of available tools. If the proxy returns 403, it refreshes the session
// token and retries once.
func (b *Bridge) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	result, err := b.doToolsList()
	if err != nil {
		b.logError("tools/list failed: %v", err)
		return &JSONRPCResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	return &JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (b *Bridge) doToolsList() (any, error) {
	httpReq, err := http.NewRequest("GET", b.proxyURL+"/tools", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+b.sessionToken)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call proxy: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		b.refreshSessionToken()

		httpReq, err = http.NewRequest("GET", b.proxyURL+"/tools", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+b.sessionToken)

		resp, err = b.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to call proxy on retry: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy returned status %d", resp.StatusCode)
	}

	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// handleToolsCall forwards a tools/call request to the proxy and returns the
// tool execution result. If the proxy returns 403, it refreshes the session
// token and retries once.
func (b *Bridge) handleToolsCall(req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("invalid params: %v", err),
			},
		}
	}

	text, err := b.doToolsCall(params)
	if err != nil {
		b.logError("tools/call failed: %v", err)
		return &JSONRPCResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": fmt.Sprintf("error: %v", err),
					},
				},
			},
		}
	}

	return &JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": text,
				},
			},
		},
	}
}

func (b *Bridge) doToolsCall(params ToolCallParams) (string, error) {
	body, err := json.Marshal(map[string]any{
		"name":      params.Name,
		"arguments": params.Arguments,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", b.proxyURL+"/call", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.sessionToken)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call proxy: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		b.refreshSessionToken()

		httpReq, err = http.NewRequest("POST", b.proxyURL+"/call", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("failed to create retry request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+b.sessionToken)

		resp, err = b.client.Do(httpReq)
		if err != nil {
			return "", fmt.Errorf("failed to call proxy on retry: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(respBody), nil
}

// refreshSessionToken re-reads the session token from the system keyring.
// This handles the case where the proxy was restarted and generated a new token.
func (b *Bridge) refreshSessionToken() {
	token, err := config.GetSessionToken()
	if err != nil {
		b.logError("failed to refresh session token: %v", err)
		return
	}
	b.sessionToken = token
}

// writeResponse marshals a JSON-RPC response and writes it to stdout with a
// trailing newline. Errors are logged to stderr.
func (b *Bridge) writeResponse(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		b.logError("failed to marshal response: %v", err)
		return
	}

	if _, err := fmt.Fprintf(b.stdout, "%s\n", data); err != nil {
		b.logError("failed to write response: %v", err)
	}
}

// logError writes an error message to stderr. Errors must never be written to
// stdout, which is reserved for the JSON-RPC transport.
func (b *Bridge) logError(msg string, args ...any) {
	fmt.Fprintf(b.stderr, msg+"\n", args...)
}
