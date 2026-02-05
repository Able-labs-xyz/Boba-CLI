<p align="center">
<pre>
██████╗  ██████╗ ██████╗  █████╗
██╔══██╗██╔═══██╗██╔══██╗██╔══██╗
██████╔╝██║   ██║██████╔╝███████║
██╔══██╗██║   ██║██╔══██╗██╔══██║
██████╔╝╚██████╔╝██████╔╝██║  ██║
╚═════╝  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝
</pre>
</p>

<p align="center">
  <img src="https://img.shields.io/npm/v/@boba/cli?color=B184F5&style=flat-square" alt="npm version" />
  <img src="https://img.shields.io/badge/node-%3E%3D18-B184F5?style=flat-square" alt="node version" />
  <img src="https://img.shields.io/badge/license-MIT-B184F5?style=flat-square" alt="license" />
</p>

<h4 align="center">Connect Claude to Boba trading in seconds.</h4>

<p align="center">
  <a href="#install">Install</a> •
  <a href="#setup">Setup</a> •
  <a href="#usage">Usage</a> •
  <a href="#commands">Commands</a>
</p>

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Able-labs-xyz/Boba-CLI/main/install.sh | bash
```

Or with npm:
```bash
npm install -g @boba/cli
```

## Setup

**1. Get your agent credentials** from [agents.boba.xyz](https://agents.boba.xyz)

**2. Initialize the CLI:**
```bash
boba init
```

**3. Configure Claude (auto):**
```bash
boba install
```

**4. Start trading:**
```bash
boba launch
```

That's it. Claude now has access to Boba trading tools.

---

## Usage

```
┌────────────────┐      ┌─────────────┐      ┌─────────────┐
│  Claude        │ ───▶ │  Boba CLI   │ ───▶ │  Boba MCP   │
│  (no creds)    │      │  (proxy)    │      │  (backend)  │
└────────────────┘      └─────────────┘      └─────────────┘
    localhost              JWT auth             trading API
```

- **Claude never sees your credentials** — only connects to localhost
- **You control access** — stop the proxy anytime
- **Full audit trail** — all tool calls are logged

---

## Commands

| Command | Description |
|---------|-------------|
| `boba init` | Set up agent credentials |
| `boba proxy` | Start the MCP proxy server |
| `boba install` | Auto-configure Claude Desktop & Code |
| `boba launch` | Start proxy + open Claude |
| `boba status` | Show connection status |
| `boba logout` | Clear credentials |

### Quick Examples

```bash
# Start proxy on custom port
boba proxy --port 4000

# Install for Claude Desktop only
boba install --desktop

# Install for Claude Code only
boba install --code

# Launch with Claude Desktop instead of Code
boba launch --desktop
```

---

## Security

- Secrets stored in OS keychain (macOS Keychain, Windows Credential Manager)
- All backend traffic over HTTPS
- Revoke access anytime at [agents.boba.xyz](https://agents.boba.xyz)

## Disclaimer

This software is experimental and provided "as is" without warranty of any kind. Use at your own risk. Boba assumes no liability for any losses, damages, or issues arising from the use of this tool. Trading involves significant risk — never trade more than you can afford to lose.

## License

MIT
