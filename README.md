# 🧋 Boba CLI

Connect AI agents (like Claude) to Boba trading without exposing credentials.

## Quick Start

```bash
# Install globally
npm install -g @boba/cli

# Initialize with your agent credentials (from agents.boba.xyz)
boba init

# Start the proxy server
boba start
```

## How It Works

```
┌──────────────┐      ┌─────────────┐      ┌───────────────┐
│Claude Desktop│─────▶│  boba-cli   │─────▶│  Hosted MCP   │
│  (no creds)  │      │ (has creds) │      │  (Railway)    │
└──────────────┘      └─────────────┘      └───────────────┘
     localhost:3456        JWT auth           Backend API
```

1. **Claude never has direct credentials** - only connects to localhost
2. **You control access** - stop the CLI anytime to revoke
3. **Full audit trail** - CLI logs all agent activity

## Commands

### `boba init`

Initialize with your agent credentials from [agents.boba.xyz](https://agents.boba.xyz).

```bash
# Interactive mode
boba init

# Or pass credentials directly
boba init --agent-id YOUR_AGENT_ID --secret YOUR_SECRET
```

### `boba start`

Start the proxy server for Claude to connect to.

```bash
# Default port 3456
boba start

# Custom port
boba start --port 3000
```

### `boba status`

Show current connection status and configuration.

```bash
boba status
```

### `boba logout`

Clear stored credentials.

```bash
boba logout
```

### `boba config`

View or update configuration.

```bash
# View current config
boba config

# Set custom MCP URL
boba config --mcp-url https://your-mcp.example.com

# Reset to defaults
boba config --reset
```

### `boba auth`

Test authentication without starting the proxy.

```bash
boba auth
```

## Claude Desktop Configuration

Add this to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "boba": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "-H", "Content-Type: application/json",
        "-d", "{\"tool\": \"get_portfolio\", \"args\": {}}",
        "http://127.0.0.1:3456/call"
      ]
    }
  }
}
```

Or use the HTTP endpoint directly with any MCP-compatible client.

## Security

- Credentials are stored securely in `~/.config/boba-cli/`
- Agent secrets are never logged
- All traffic to the backend is over HTTPS
- You can revoke agent access anytime at agents.boba.xyz

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run in development
npm run dev
```

## License

MIT
