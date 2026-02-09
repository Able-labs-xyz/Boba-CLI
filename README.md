<p align="center">
  <img src="./assets/logo.svg" alt="BOBA AGENTS" width="400" />
</p>

<p align="center">
  <img src="https://img.shields.io/npm/v/@tradeboba/cli?color=B184F5&style=flat-square" alt="npm" />
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-B184F5?style=flat-square" alt="platform" />
  <img src="https://img.shields.io/badge/license-MIT-B184F5?style=flat-square" alt="license" />
</p>

<p align="center">
  <b>Connect Claude to Boba trading in seconds.</b>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#quick-start">Quick Start</a> ·
  <a href="#commands">Commands</a> ·
  <a href="#how-it-works">How It Works</a>
</p>

<br />

## Install

```bash
npm install -g @tradeboba/cli
```

<br />

## Quick Start

```bash
npm install -g @tradeboba/cli
boba
```

That's it — the interactive menu walks you through everything.

<br />

## Commands

| Command | Description |
|:--------|:------------|
| `boba login` | Log in with your agent credentials |
| `boba install` | Set up Claude to use Boba |
| `boba launch` | Start trading with Claude |
| `boba start` | Run the Boba proxy |
| `boba status` | See if everything's working |
| `boba config` | Change your settings |
| `boba auth` | Test your connection |
| `boba logout` | Sign out |

<details>
<summary>Command options</summary>

```bash
boba login --agent-id ID --secret S   # Non-interactive login
boba start --port 4000                 # Custom port
boba install --desktop-only            # Claude Desktop only
boba install --code-only               # Claude Code only
boba launch --iterm                    # Use iTerm instead of Terminal (macOS)
```

</details>

<br />

## How It Works

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Claude    │ ───▶ │  Boba CLI   │ ───▶ │  Boba MCP   │
│  (no creds) │      │   (proxy)   │      │  (backend)  │
└─────────────┘      └─────────────┘      └─────────────┘
   localhost            JWT auth           trading API
```

- Claude never sees your credentials
- You control access — stop the proxy anytime
- Full audit trail — all tool calls logged

<br />

## Upgrading to v0.3.0

> [!IMPORTANT]
> v0.3.0 is a full rewrite — the CLI is now a native Go binary shipped through npm. It's faster, has no Node.js runtime dependency, and includes an interactive TUI.
>
> **What changed:**
> - `boba init` is now `boba login` (`boba init` still works as an alias)
> - Interactive menus — just run `boba` to see all options
> - `boba launch` now lets you pick Claude Code or Claude Desktop
> - `boba launch` now lets you launch with layouts
> - `boba login` now has onboarding flow 
> - Nice TUI changes using BubbleTea, LipGloss and HuH
>
> **To upgrade:**
> ```bash
> npm install -g @tradeboba/cli
> boba login
> ```

<br />

## Security

| | |
|:--|:--|
| **Credential Storage** | Agent secret + auth tokens stored in OS Keychain |
| **Proxy Auth** | Per-session token — only the MCP bridge can call the proxy |
| **Transport** | HTTPS enforced for all backend communication |
| **URL Allowlisting** | Backend URLs restricted to known Boba hosts |
| **No Debug Logging** | No sensitive data written to logs |
| **Access Control** | Revoke anytime at [agents.boba.xyz](https://agents.boba.xyz) |

<br />

## License

MIT

<br />

---

<br />

> [!WARNING]
> **This software is experimental.** Provided "as is" without warranty of any kind.
>
> **Use at your own risk.** Boba assumes no liability for any losses or damages arising from use of this software.
>
> **Trading involves significant financial risk.** Never trade more than you can afford to lose.
