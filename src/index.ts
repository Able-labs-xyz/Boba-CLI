#!/usr/bin/env node

import { Command } from 'commander';
import { config } from './config.js';
import { authenticate, ensureAuthenticated } from './auth.js';
import { createProxyServer } from './proxy.js';
import { logger, colors } from './logger.js';
import { printStartup, printBanner, matcha, matchaDim, matchaBright, bouncyLoader, bubbleLoader, BOBA_MINI, bobaFrames, colorizeFrame, createSpinner, withSpinner } from './art.js';
import * as readline from 'readline';

const program = new Command();

// Custom help display
function showHelp() {
  console.clear();
  console.log(colorizeFrame(bobaFrames[1]));
  console.log();
  console.log(matchaBright('  ╔════════════════════════════════════════════╗'));
  console.log(matchaBright('  ║') + matcha('           🧋 BOBA AGENT CLI               ') + matchaBright('║'));
  console.log(matchaBright('  ║') + matchaDim('     Connect AI agents to Boba trading     ') + matchaBright('║'));
  console.log(matchaBright('  ╚════════════════════════════════════════════╝'));
  console.log();
  console.log(matchaBright('  COMMANDS'));
  console.log(matchaDim('  ─────────────────────────────────────────────'));
  console.log(`  ${matcha('init')}      ${matchaDim('Initialize with your agent credentials')}`);
  console.log(`  ${matcha('install')}   ${matchaDim('Install MCP for Claude Desktop/Code')}`);
  console.log(`  ${matcha('launch')}    ${matchaDim('Start proxy + open Claude (one command!)')}`);
  console.log(`  ${matcha('start')}     ${matchaDim('Start the proxy server for Claude')}`);
  console.log(`  ${matcha('status')}    ${matchaDim('Show current connection status')}`);
  console.log(`  ${matcha('config')}    ${matchaDim('View or update configuration')}`);
  console.log(`  ${matcha('logout')}    ${matchaDim('Clear stored credentials')}`);
  console.log();
  console.log(matchaBright('  OPTIONS'));
  console.log(matchaDim('  ─────────────────────────────────────────────'));
  console.log(`  ${matcha('-v, --version')}    ${matchaDim('Show version number')}`);
  console.log(`  ${matcha('-h, --help')}       ${matchaDim('Show this help menu')}`);
  console.log();
  console.log(matchaBright('  QUICK START'));
  console.log(matchaDim('  ─────────────────────────────────────────────'));
  console.log(`  ${matchaDim('1.')} ${matcha('boba init')}       ${matchaDim('Set up credentials')}`);
  console.log(`  ${matchaDim('2.')} ${matcha('boba install')}    ${matchaDim('Configure Claude Desktop/Code')}`);
  console.log(`  ${matchaDim('3.')} ${matcha('boba launch')}     ${matchaDim('Start everything!')}`);
  console.log();
}

program
  .name('boba')
  .description('🧋 Boba Agent CLI - Connect AI agents to Boba trading')
  .version('0.1.0')
  .helpOption('-h, --help', 'Show help menu')
  .addHelpCommand(false)
  .action(() => {
    showHelp();
  });

program.on('--help', () => {
  // Override default help
});

program.configureHelp({
  formatHelp: () => {
    showHelp();
    return '';
  }
});

// ============================================================================
// INIT COMMAND
// ============================================================================
program
  .command('init')
  .description('Initialize with your agent credentials')
  .option('-i, --agent-id <id>', 'Agent ID')
  .option('-s, --secret <secret>', 'Agent secret')
  .option('-n, --name <name>', 'Agent name (optional)')
  .action(async (options) => {
    printStartup();

    let agentId = options.agentId;
    let agentSecret = options.secret;
    let agentName = options.name;

    // Interactive mode if credentials not provided
    if (!agentId || !agentSecret) {
      console.log(`  ${matcha('Enter your agent credentials from agents.boba.xyz')}\n`);

      const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
      });

      const question = (prompt: string): Promise<string> => {
        return new Promise((resolve) => {
          rl.question(`  ${matcha('›')} ${prompt}: `, resolve);
        });
      };

      if (!agentId) {
        agentId = await question('Agent ID');
      }
      if (!agentSecret) {
        agentSecret = await question('Agent Secret');
      }
      if (!agentName) {
        agentName = await question('Agent Name (optional)');
      }

      rl.close();
      console.log();
    }

    if (!agentId || !agentSecret) {
      logger.error('Agent ID and secret are required');
      process.exit(1);
    }

    // Save credentials (secret goes to OS keychain)
    await config.setCredentials({
      agentId,
      agentSecret,
      name: agentName || undefined,
    });

    // Test authentication with spinner
    const spinner = createSpinner('Verifying credentials', 'dots');
    spinner.start();

    const tokens = await authenticate();

    if (tokens) {
      spinner.succeed('Credentials verified!');
    } else {
      spinner.fail('Verification failed');
    }

    if (tokens) {
      console.log();
      logger.success('Credentials verified and saved!');
      console.log();
      console.log(`  ${matchaDim('Agent:')}     ${matchaBright(tokens.agentName)}`);
      console.log(`  ${matchaDim('ID:')}        ${matchaDim(tokens.agentId.slice(0, 8) + '...')}`);
      console.log(`  ${matchaDim('EVM:')}       ${matchaDim(tokens.evmAddress)}`);
      if (tokens.solanaAddress) {
        console.log(`  ${matchaDim('Solana:')}    ${matchaDim(tokens.solanaAddress)}`);
      }
      console.log();
      console.log(`  ${matcha('Run')} ${matchaBright('boba start')} ${matcha('to start the proxy server')}`);
      console.log();
    } else {
      await config.clearCredentials();
      logger.error('Failed to verify credentials. Please check and try again.');
      process.exit(1);
    }
  });

// ============================================================================
// START COMMAND
// ============================================================================
program
  .command('start')
  .description('Start the proxy server for Claude')
  .option('-p, --port <port>', 'Port to run on', '3456')
  .action(async (options) => {
    printStartup();

    if (!config.hasCredentials()) {
      logger.error('No credentials found. Run "boba init" first.');
      process.exit(1);
    }

    const port = parseInt(options.port);
    if (port) {
      config.setProxyPort(port);
    }

    try {
      // Show spinner while connecting
      const spinner = createSpinner('Starting proxy server', 'dots');
      spinner.start();

      const proxy = createProxyServer();

      spinner.succeed('Proxy server ready');

      // Handle shutdown
      process.on('SIGINT', () => {
        console.log();
        logger.info('Shutting down...');
        proxy.stop();
        process.exit(0);
      });

      process.on('SIGTERM', () => {
        proxy.stop();
        process.exit(0);
      });

      await proxy.start();
    } catch (error: any) {
      logger.error(error.message);
      process.exit(1);
    }
  });

// ============================================================================
// STATUS COMMAND
// ============================================================================
program
  .command('status')
  .description('Show current connection status')
  .action(async () => {
    console.clear();
    console.log(colorizeFrame(bobaFrames[1]));
    console.log();

    const creds = await config.getCredentials();
    const tokens = config.getTokens();

    // Apply color to full 44-char string to avoid counting issues
    const B = matchaBright;
    const G = matcha;
    const D = matchaDim;
    const W = colors.warning;

    // Helper: create exactly 44 char content, then colorize
    const make = (str: string) => str.padEnd(44).slice(0, 44);

    console.log(B('  ╔════════════════════════════════════════════╗'));
    console.log(B('  ║') + G(make('           BOBA CONNECTION STATUS           ')) + B('║'));
    console.log(B('  ╠════════════════════════════════════════════╣'));

    if (!creds) {
      console.log(B('  ║') + W(make('    Status:      Not Initialized           ')) + B('║'));
      console.log(B('  ╠════════════════════════════════════════════╣'));
      console.log(B('  ║') + D(make('    Run "boba init" to set up              ')) + B('║'));
    } else {
      const agentId = creds.agentId.slice(0, 25).padEnd(25);
      console.log(B('  ║') + '    Credentials: ' + G('Configured') + '                 ' + B('║'));
      console.log(B('  ║') + D(make(`    Agent ID:    ${agentId}`)) + B('║'));

      if (tokens) {
        const isExpired = config.isTokenExpired();
        console.log(B('  ╠════════════════════════════════════════════╣'));

        if (isExpired) {
          console.log(B('  ║') + '    Token:       ' + W('Expired') + '                    ' + B('║'));
        } else {
          console.log(B('  ║') + '    Token:       ' + G('Valid') + '                      ' + B('║'));
        }

        const agent = tokens.agentName.slice(0, 25).padEnd(25);
        const evm = tokens.evmAddress.slice(0, 25).padEnd(25);
        console.log(B('  ║') + D(make(`    Agent:       ${agent}`)) + B('║'));
        console.log(B('  ║') + D(make(`    EVM:         ${evm}`)) + B('║'));
        if (tokens.solanaAddress) {
          const sol = tokens.solanaAddress.slice(0, 25).padEnd(25);
          console.log(B('  ║') + D(make(`    Solana:      ${sol}`)) + B('║'));
        }
      } else {
        console.log(B('  ╠════════════════════════════════════════════╣'));
        console.log(B('  ║') + '    Token:       ' + W('Not authenticated') + '          ' + B('║'));
      }
    }

    console.log(B('  ╠════════════════════════════════════════════╣'));
    console.log(B('  ║') + D(make('              CONFIGURATION                 ')) + B('║'));
    console.log(B('  ╠════════════════════════════════════════════╣'));
    const mcp = config.getMcpUrl().replace('https://', '').slice(0, 25).padEnd(25);
    const port = String(config.getProxyPort()).padEnd(25);
    console.log(B('  ║') + D(make(`    MCP:         ${mcp}`)) + B('║'));
    console.log(B('  ║') + D(make(`    Proxy Port:  ${port}`)) + B('║'));
    console.log(B('  ╚════════════════════════════════════════════╝'));
    console.log();
  });

// ============================================================================
// LOGOUT COMMAND
// ============================================================================
program
  .command('logout')
  .description('Clear stored credentials')
  .action(async () => {
    printBanner();
    console.log();

    await config.clearCredentials();
    logger.success('Logged out. Credentials cleared from keychain.');
    console.log();
  });

// ============================================================================
// CONFIG COMMAND
// ============================================================================
program
  .command('config')
  .description('View or update configuration')
  .option('--mcp-url <url>', 'Set MCP server URL')
  .option('--auth-url <url>', 'Set auth server URL')
  .option('--port <port>', 'Set default proxy port')
  .option('--reset', 'Reset all config to defaults')
  .action(async (options) => {
    const B = matchaBright;
    const G = matcha;
    const D = matchaDim;
    const make = (str: string) => str.padEnd(44).slice(0, 44);

    const showUpdateBox = (title: string, value: string) => {
      console.clear();
      console.log(colorizeFrame(bobaFrames[1]));
      console.log();
      console.log(B('  ╔════════════════════════════════════════════╗'));
      console.log(B('  ║') + G(make('            CONFIG UPDATED ✓                ')) + B('║'));
      console.log(B('  ╠════════════════════════════════════════════╣'));
      console.log(B('  ║') + D(make(`    ${title}`)) + B('║'));
      console.log(B('  ║') + G(make(`    ${value.slice(0, 40)}`)) + B('║'));
      console.log(B('  ╚════════════════════════════════════════════╝'));
      console.log();
    };

    if (options.reset) {
      config.reset();
      showUpdateBox('Reset to defaults:', 'All settings restored');
      return;
    }

    if (options.mcpUrl) {
      config.setMcpUrl(options.mcpUrl);
      showUpdateBox('MCP URL:', options.mcpUrl);
    }

    if (options.authUrl) {
      config.setAuthUrl(options.authUrl);
      showUpdateBox('Auth URL:', options.authUrl);
    }

    if (options.port) {
      config.setProxyPort(parseInt(options.port));
      showUpdateBox('Proxy Port:', options.port);
    }

    // Show current config if no options provided
    if (!options.mcpUrl && !options.authUrl && !options.port && !options.reset) {
      console.clear();
      console.log(colorizeFrame(bobaFrames[1]));
      console.log();

      const cfg = config.getAll();

      console.log(B('  ╔════════════════════════════════════════════╗'));
      console.log(B('  ║') + G(make('              CONFIGURATION                 ')) + B('║'));
      console.log(B('  ╠════════════════════════════════════════════╣'));

      const mcpUrl = cfg.mcpUrl.replace('https://', '').slice(0, 25).padEnd(25);
      const authUrl = cfg.authUrl.replace('https://', '').slice(0, 25).padEnd(25);
      const port = String(cfg.proxyPort).padEnd(25);
      const logLevel = cfg.logLevel.padEnd(25);

      console.log(B('  ║') + D(make(`    MCP URL:     ${mcpUrl}`)) + B('║'));
      console.log(B('  ║') + D(make(`    Auth URL:    ${authUrl}`)) + B('║'));
      console.log(B('  ║') + D(make(`    Proxy Port:  ${port}`)) + B('║'));
      console.log(B('  ║') + D(make(`    Log Level:   ${logLevel}`)) + B('║'));

      console.log(B('  ╠════════════════════════════════════════════╣'));
      const configPath = config.getConfigPath();
      const shortPath = configPath.length > 40 ? '~/' + configPath.split('/').slice(-2).join('/') : configPath;
      console.log(B('  ║') + D(make(`    File: ${shortPath.slice(0, 33)}`)) + B('║'));
      console.log(B('  ╚════════════════════════════════════════════╝'));
    }

    console.log();
  });

// ============================================================================
// AUTH COMMAND (for testing)
// ============================================================================
program
  .command('auth')
  .description('Test authentication and refresh tokens')
  .action(async () => {
    printBanner();
    console.log();

    if (!config.hasCredentials()) {
      logger.error('No credentials found. Run "boba init" first.');
      process.exit(1);
    }

    const spinner = createSpinner('Authenticating', 'dots');
    spinner.start();

    const tokens = await authenticate();

    if (tokens) {
      spinner.succeed('Authentication successful!');
      console.log();
      console.log(`  ${matchaDim('Agent:')}     ${matchaBright(tokens.agentName)}`);
      console.log(`  ${matchaDim('Agent ID:')} ${matchaDim(tokens.agentId)}`);
      console.log(`  ${matchaDim('EVM:')}       ${matchaDim(tokens.evmAddress)}`);
      console.log(`  ${matchaDim('Expires:')}   ${matchaDim(tokens.accessTokenExpiresAt)}`);
      console.log();
    } else {
      spinner.fail('Authentication failed');
      process.exit(1);
    }
  });

// ============================================================================
// INSTALL COMMAND (configure Claude Desktop & Claude Code)
// ============================================================================
program
  .command('install')
  .description('Install boba MCP server for Claude Desktop and Claude Code')
  .option('--desktop-only', 'Only install for Claude Desktop')
  .option('--code-only', 'Only install for Claude Code')
  .action(async (options) => {
    printBanner();
    console.log();

    const os = await import('os');
    const fs = await import('fs');
    const path = await import('path');
    const { execSync } = await import('child_process');

    const homeDir = os.homedir();
    const platform = os.platform();

    // Find the boba command - works whether installed globally or locally
    let bobaCommand: string;
    let bobaArgs: string[];
    let installMethod: string;

    try {
      // First, check if boba is in PATH
      if (platform === 'win32') {
        execSync('where boba', { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] });
      } else {
        execSync('which boba', { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] });
      }
      // If found, just use "boba" - the system will resolve the path
      bobaCommand = 'boba';
      bobaArgs = ['mcp'];
      installMethod = 'global';
    } catch {
      // Fallback: use npx to run boba (works if published to npm)
      try {
        if (platform === 'win32') {
          execSync('where npx', { stdio: ['pipe', 'pipe', 'pipe'] });
        } else {
          execSync('which npx', { stdio: ['pipe', 'pipe', 'pipe'] });
        }
        bobaCommand = 'npx';
        bobaArgs = ['-y', '@tradeboba/cli', 'mcp'];
        installMethod = 'npx';
      } catch {
        // Last resort: use node with the current script path
        const nodeExecutable = process.execPath;
        let cliPath: string;

        try {
          const url = new URL(import.meta.url);
          cliPath = path.join(path.dirname(url.pathname.replace(/^\/([A-Z]:)/i, '$1')), 'index.js');
        } catch {
          cliPath = __filename;
        }

        bobaCommand = nodeExecutable;
        bobaArgs = [cliPath, 'mcp'];
        installMethod = 'local';
      }
    }

    const bobaConfig = {
      command: bobaCommand,
      args: bobaArgs,
    };

    const results: { target: string; status: 'installed' | 'updated' | 'skipped' | 'error'; message?: string }[] = [];

    // Claude Desktop config paths by platform
    let desktopConfigPath: string;
    if (platform === 'darwin') {
      desktopConfigPath = path.join(homeDir, 'Library', 'Application Support', 'Claude', 'claude_desktop_config.json');
    } else if (platform === 'win32') {
      desktopConfigPath = path.join(homeDir, 'AppData', 'Roaming', 'Claude', 'claude_desktop_config.json');
    } else {
      desktopConfigPath = path.join(homeDir, '.config', 'claude', 'claude_desktop_config.json');
    }

    // Claude Code config path
    const codeConfigPath = path.join(homeDir, '.claude.json');

    // Install for Claude Desktop
    if (!options.codeOnly) {
      try {
        let desktopConfig: any = { mcpServers: {} };

        if (fs.existsSync(desktopConfigPath)) {
          const content = fs.readFileSync(desktopConfigPath, 'utf-8');
          desktopConfig = JSON.parse(content);
          if (!desktopConfig.mcpServers) {
            desktopConfig.mcpServers = {};
          }
        } else {
          // Create directory if it doesn't exist
          const configDir = path.dirname(desktopConfigPath);
          if (!fs.existsSync(configDir)) {
            fs.mkdirSync(configDir, { recursive: true });
          }
        }

        const hadBoba = !!desktopConfig.mcpServers.boba;
        desktopConfig.mcpServers.boba = bobaConfig;

        fs.writeFileSync(desktopConfigPath, JSON.stringify(desktopConfig, null, 2));
        results.push({
          target: 'Claude Desktop',
          status: hadBoba ? 'updated' : 'installed',
        });
      } catch (err: any) {
        results.push({
          target: 'Claude Desktop',
          status: 'error',
          message: err.message,
        });
      }
    }

    // Install for Claude Code
    if (!options.desktopOnly) {
      try {
        let codeConfig: any = { projects: {} };

        if (fs.existsSync(codeConfigPath)) {
          const content = fs.readFileSync(codeConfigPath, 'utf-8');
          codeConfig = JSON.parse(content);
          if (!codeConfig.projects) {
            codeConfig.projects = {};
          }
        }

        // Add to the home directory project (global fallback)
        if (!codeConfig.projects[homeDir]) {
          codeConfig.projects[homeDir] = {
            allowedTools: [],
            mcpContextUris: [],
            mcpServers: {},
            enabledMcpjsonServers: [],
            disabledMcpjsonServers: [],
          };
        }

        if (!codeConfig.projects[homeDir].mcpServers) {
          codeConfig.projects[homeDir].mcpServers = {};
        }

        const hadBoba = !!codeConfig.projects[homeDir].mcpServers.boba;
        codeConfig.projects[homeDir].mcpServers.boba = bobaConfig;

        fs.writeFileSync(codeConfigPath, JSON.stringify(codeConfig, null, 2));
        results.push({
          target: 'Claude Code (global)',
          status: hadBoba ? 'updated' : 'installed',
        });
      } catch (err: any) {
        results.push({
          target: 'Claude Code',
          status: 'error',
          message: err.message,
        });
      }
    }

    // Display results
    const B = matchaBright;
    const G = matcha;
    const D = matchaDim;
    const W = colors.warning;
    const make = (str: string) => str.padEnd(44).slice(0, 44);

    console.log(B('  ╔════════════════════════════════════════════╗'));
    console.log(B('  ║') + G(make('          MCP INSTALLATION COMPLETE         ')) + B('║'));
    console.log(B('  ╠════════════════════════════════════════════╣'));

    for (const result of results) {
      let statusIcon: string;
      let statusText: string;

      if (result.status === 'installed') {
        statusIcon = G('✓');
        statusText = G('Installed');
      } else if (result.status === 'updated') {
        statusIcon = G('✓');
        statusText = G('Updated');
      } else if (result.status === 'skipped') {
        statusIcon = D('○');
        statusText = D('Skipped');
      } else {
        statusIcon = W('✗');
        statusText = W('Error');
      }

      console.log(B('  ║') + `  ${statusIcon} ${result.target.padEnd(20)} ${statusText.padEnd(15)}` + B('║'));
      if (result.message) {
        console.log(B('  ║') + D(make(`      ${result.message.slice(0, 38)}`)) + B('║'));
      }
    }

    console.log(B('  ╠════════════════════════════════════════════╣'));
    console.log(B('  ║') + D(make('              CONFIGURATION                 ')) + B('║'));
    console.log(B('  ╠════════════════════════════════════════════╣'));
    const methodLabel = installMethod === 'global' ? 'Global (boba)' : installMethod === 'npx' ? 'NPX (@tradeboba/cli)' : 'Local (node)';
    console.log(B('  ║') + D(make(`  Method: ${methodLabel}`)) + B('║'));
    const cmdDisplay = `${bobaCommand} ${bobaArgs.join(' ')}`;
    console.log(B('  ║') + D(make(`  Command: ${cmdDisplay.slice(0, 33)}`)) + B('║'));
    console.log(B('  ╠════════════════════════════════════════════╣'));
    console.log(B('  ║') + D(make('                NEXT STEPS                  ')) + B('║'));
    console.log(B('  ╠════════════════════════════════════════════╣'));
    console.log(B('  ║') + D(make('  1. Run: boba init  (if not done)          ')) + B('║'));
    console.log(B('  ║') + D(make('  2. Run: boba start                        ')) + B('║'));
    console.log(B('  ║') + D(make('  3. Restart Claude Desktop/Code            ')) + B('║'));
    console.log(B('  ║') + D(make('  4. Boba tools will be available!          ')) + B('║'));
    console.log(B('  ╚════════════════════════════════════════════╝'));
    console.log();
  });

// ============================================================================
// LAUNCH COMMAND (start proxy in new terminal + open Claude)
// ============================================================================
program
  .command('launch')
  .description('Launch boba proxy and open Claude (macOS)')
  .option('--desktop', 'Open Claude Desktop instead of Claude Code')
  .option('--iterm', 'Use iTerm instead of Terminal.app')
  .action(async (options) => {
    const os = await import('os');
    const { execSync, spawn } = await import('child_process');

    const platform = os.platform();

    if (platform !== 'darwin') {
      logger.error('Launch command currently only supports macOS');
      logger.info('On other platforms, run "boba start" manually then open Claude');
      process.exit(1);
    }

    if (!config.hasCredentials()) {
      logger.error('No credentials found. Run "boba init" first.');
      process.exit(1);
    }

    printBanner();
    console.log();

    const B = matchaBright;
    const G = matcha;
    const D = matchaDim;

    console.log(`  ${G('●')} Launching boba proxy...`);

    // Get the boba command path
    const bobaPath = process.argv[1]; // Path to current script
    const nodePath = process.execPath;

    try {
      if (options.iterm) {
        // Use iTerm2
        const script = `
          tell application "iTerm"
            activate
            set newWindow to (create window with default profile)
            tell current session of newWindow
              write text "${nodePath} ${bobaPath} start"
            end tell
          end tell
        `;
        execSync(`osascript -e '${script.replace(/'/g, "'\"'\"'")}'`);
      } else {
        // Use Terminal.app
        const script = `
          tell application "Terminal"
            activate
            do script "${nodePath} \\"${bobaPath}\\" start"
          end tell
        `;
        execSync(`osascript -e '${script.replace(/'/g, "'\"'\"'")}'`);
      }

      console.log(`  ${G('✓')} Proxy starting in new terminal window`);

      // Wait a moment for proxy to start
      await new Promise(resolve => setTimeout(resolve, 2000));

      if (options.desktop) {
        // Open Claude Desktop
        console.log(`  ${G('●')} Opening Claude Desktop...`);
        try {
          execSync('open -a "Claude"');
          console.log(`  ${G('✓')} Claude Desktop launched`);
        } catch {
          logger.warning('Could not open Claude Desktop. Make sure it\'s installed.');
        }
      } else {
        // Open Claude Code in a new terminal window (needs TTY)
        console.log(`  ${G('●')} Opening Claude Code...`);
        const cwd = process.cwd();
        try {
          if (options.iterm) {
            const script = `
              tell application "iTerm"
                activate
                set newWindow to (create window with default profile)
                tell current session of newWindow
                  write text "cd '${cwd}' && claude"
                end tell
              end tell
            `;
            execSync(`osascript -e '${script.replace(/'/g, "'\"'\"'")}'`);
          } else {
            const script = `
              tell application "Terminal"
                activate
                do script "cd '${cwd}' && claude"
              end tell
            `;
            execSync(`osascript -e '${script.replace(/'/g, "'\"'\"'")}'`);
          }
          console.log(`  ${G('✓')} Claude Code launched in new terminal`);
        } catch (err: any) {
          logger.warning('Could not launch Claude Code: ' + err.message);
          logger.info('Run "claude" manually in your terminal');
        }
      }

      console.log();
      console.log(B('  ╔════════════════════════════════════════════╗'));
      console.log(B('  ║') + G('            BOBA IS READY! 🧋               ') + B('║'));
      console.log(B('  ╠════════════════════════════════════════════╣'));
      console.log(B('  ║') + D('  Proxy running in separate terminal        ') + B('║'));
      console.log(B('  ║') + D('  Claude is open and ready to use           ') + B('║'));
      console.log(B('  ║') + D('                                            ') + B('║'));
      console.log(B('  ║') + D('  Try: "Show me my portfolio"               ') + B('║'));
      console.log(B('  ╚════════════════════════════════════════════╝'));
      console.log();

    } catch (err: any) {
      logger.error(`Failed to launch: ${err.message}`);
      process.exit(1);
    }
  });

// ============================================================================
// MCP COMMAND (for Claude Desktop stdio integration)
// ============================================================================
program
  .command('mcp')
  .description('Run as MCP server for Claude Desktop (stdio mode)')
  .action(async () => {
    // No visual output in MCP mode - pure JSON-RPC over stdio
    if (!config.hasCredentials()) {
      const error = {
        jsonrpc: '2.0',
        id: null,
        error: { code: -32603, message: 'No credentials. Run "boba init" first.' },
      };
      console.log(JSON.stringify(error));
      process.exit(1);
    }

    const port = config.getProxyPort();
    const proxyUrl = `http://127.0.0.1:${port}`;

    // Check if proxy is running
    try {
      const axios = (await import('axios')).default;
      await axios.get(`${proxyUrl}/health`, { timeout: 2000 });
    } catch {
      const error = {
        jsonrpc: '2.0',
        id: null,
        error: { code: -32603, message: `Proxy not running. Start it with "boba start" first.` },
      };
      console.log(JSON.stringify(error));
      process.exit(1);
    }

    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      terminal: false,
    });

    // Handle JSON-RPC requests from Claude
    rl.on('line', async (line) => {
      try {
        const request = JSON.parse(line);
        const { id, method, params } = request;

        if (method === 'initialize') {
          // MCP initialization
          const response = {
            jsonrpc: '2.0',
            id,
            result: {
              protocolVersion: '2024-11-05',
              capabilities: { tools: {} },
              serverInfo: { name: 'boba', version: '0.1.0' },
            },
          };
          console.log(JSON.stringify(response));
        } else if (method === 'notifications/initialized') {
          // No response needed for notifications
        } else if (method === 'tools/list') {
          // Get tools from proxy
          try {
            const axios = (await import('axios')).default;
            const toolsRes = await axios.get(`${proxyUrl}/tools`);
            const response = {
              jsonrpc: '2.0',
              id,
              result: { tools: toolsRes.data.tools || toolsRes.data },
            };
            console.log(JSON.stringify(response));
          } catch (err: any) {
            const response = {
              jsonrpc: '2.0',
              id,
              error: { code: -32603, message: err.message },
            };
            console.log(JSON.stringify(response));
          }
        } else if (method === 'tools/call') {
          // Forward tool call to proxy
          const { name, arguments: args } = params;
          try {
            const axios = (await import('axios')).default;
            const result = await axios.post(`${proxyUrl}/call`, {
              tool: name,
              args: args || {},
            });
            const response = {
              jsonrpc: '2.0',
              id,
              result: {
                content: [{ type: 'text', text: JSON.stringify(result.data, null, 2) }],
              },
            };
            console.log(JSON.stringify(response));
          } catch (err: any) {
            const errorMsg = err.response?.data?.error || err.message;
            const response = {
              jsonrpc: '2.0',
              id,
              result: {
                content: [{ type: 'text', text: `Error: ${errorMsg}` }],
                isError: true,
              },
            };
            console.log(JSON.stringify(response));
          }
        } else {
          // Unknown method
          const response = {
            jsonrpc: '2.0',
            id,
            error: { code: -32601, message: `Unknown method: ${method}` },
          };
          console.log(JSON.stringify(response));
        }
      } catch (err: any) {
        // Parse error
        const response = {
          jsonrpc: '2.0',
          id: null,
          error: { code: -32700, message: 'Parse error' },
        };
        console.log(JSON.stringify(response));
      }
    });

    // Keep process alive
    process.stdin.resume();
  });

// Parse and run
program.parse();
