import express from 'express';
import axios from 'axios';
import ora from 'ora';
import chalk from 'chalk';
import { config } from './config.js';
import { authenticate, ensureAuthenticated, getAccessToken, refreshTokens } from './auth.js';
import { logger } from './logger.js';
import { matcha, matchaDim, matchaBright, bobaFrames, colorizeFrame } from './art.js';
import { formatToolResult } from './formatter.js';
// Spinner for tool calls
const boba = chalk.hex('#B184F5');
const bobaDim = chalk.hex('#8A5FD1');
function createToolSpinner(tool, action) {
    return ora({
        text: `${boba(tool)} ${bobaDim(action)}`,
        spinner: {
            interval: 80,
            frames: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'].map(f => boba(f)),
        },
        color: 'magenta',
    });
}
function formatDuration(ms) {
    if (ms < 1000)
        return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
}
// Tool descriptions for prettier logging
const toolDescriptions = {
    get_portfolio: 'Fetching portfolio...',
    get_portfolio_pnl: 'Getting P&L data...',
    get_swap_price: 'Getting swap quote...',
    execute_swap: 'Executing trade...',
    get_token_info: 'Looking up token...',
    search_tokens: 'Searching tokens...',
    get_trending_tokens: 'Getting trending...',
    get_brewing_status: 'Checking new launches...',
    get_recent_launches: 'Getting recent launches...',
    stream_launches: 'Streaming launches...',
    get_wallet_balance: 'Checking wallet...',
    create_limit_order: 'Creating limit order...',
    get_limit_orders: 'Getting limit orders...',
};
function getToolDescription(tool) {
    return toolDescriptions[tool] || `Calling ${tool}...`;
}
function formatResult(tool, result) {
    try {
        // Format based on tool type
        if (tool === 'get_portfolio') {
            const total = result.total_value_usd;
            return total ? `$${Number(total).toLocaleString()}` : 'Portfolio loaded';
        }
        if (tool === 'get_swap_price' || tool === 'get_swap_quote') {
            return result.dst_amount ? `Quote: ${result.dst_amount}` : 'Quote received';
        }
        if (tool === 'execute_swap') {
            return result.tx_hash ? `TX: ${result.tx_hash.slice(0, 10)}...` : 'Swap executed';
        }
        if (tool === 'get_token_info') {
            return result.symbol ? `${result.symbol} - $${result.price}` : 'Token info loaded';
        }
        if (tool === 'search_tokens') {
            const count = Array.isArray(result) ? result.length : result.tokens?.length || 0;
            return `Found ${count} tokens`;
        }
        if (tool === 'get_trending_tokens') {
            const count = Array.isArray(result) ? result.length : 0;
            return `${count} trending tokens`;
        }
        if (tool.includes('brewing') || tool.includes('launches')) {
            const count = result.events?.length || result.launches?.length || 0;
            return count > 0 ? `${count} new launches` : 'No new launches';
        }
        // Default: show first key-value or array length
        if (Array.isArray(result)) {
            return `${result.length} items`;
        }
        const keys = Object.keys(result || {});
        if (keys.length > 0) {
            const firstKey = keys[0];
            const firstValue = result[firstKey];
            if (typeof firstValue === 'string' || typeof firstValue === 'number') {
                return `${firstKey}: ${String(firstValue).slice(0, 30)}`;
            }
        }
        return 'OK';
    }
    catch {
        return 'OK';
    }
}
export function createProxyServer() {
    const app = express();
    let server = null;
    let requestCount = 0;
    app.use(express.json({ limit: '10mb' }));
    // Health check
    app.get('/health', (req, res) => {
        const tokens = config.getTokens();
        res.json({
            status: 'ok',
            agent: tokens?.agentName || 'Not authenticated',
            agentId: tokens?.agentId?.slice(0, 8) || 'N/A',
            requests: requestCount,
        });
    });
    // List tools (proxy to MCP)
    app.get('/tools', async (req, res) => {
        try {
            const mcpUrl = config.getMcpUrl();
            const response = await axios.get(`${mcpUrl}/tools`);
            res.json(response.data);
        }
        catch (error) {
            logger.error(`Failed to list tools: ${error.message}`);
            res.status(500).json({ error: 'Failed to list tools' });
        }
    });
    // Main tool call endpoint
    app.post('/call', async (req, res) => {
        const { tool, args: rawArgs } = req.body;
        const startTime = Date.now();
        requestCount++;
        if (!tool) {
            res.status(400).json({ error: 'Missing tool name' });
            return;
        }
        // Get current session info for auto-filling
        const tokens = config.getTokens();
        const agentId = tokens?.agentId;
        const evmAddress = tokens?.evmAddress;
        const solanaAddress = tokens?.solanaAddress;
        // Tools that require user_id to be auto-filled (always replace, even fake IDs)
        const userIdTools = [
            'get_portfolio',
            'get_portfolio_summary',
            'get_portfolio_pnl',
            'get_trade_history',
            'get_pnl_chart',
            'get_user_xp',
            'get_transfers',
            'get_wallet_balance',
            'get_limit_orders',
            'get_dca_orders',
            'get_twap_orders',
            'get_positions',
            'create_limit_order',
            'cancel_limit_order',
            'get_user_swaps',
            'refresh_native_balances',
            'start_portfolio_stream',
            'get_portfolio_price_updates',
            'stop_portfolio_stream',
        ];
        // Helper to check if a value looks like a fake/placeholder ID or wrong type
        const isFakeId = (id) => {
            if (!id)
                return true;
            // Common fake patterns Claude uses
            if (/^1+$/.test(id))
                return true; // "111111..."
            if (/^0+$/.test(id))
                return true; // "000000..."
            if (id === 'me' || id === 'self')
                return true;
            // Wallet addresses (Claude sometimes confuses wallet with user_id)
            if (id.startsWith('0x') && id.length === 42)
                return true; // EVM address
            if (id.length >= 32 && id.length <= 44 && /^[1-9A-HJ-NP-Za-km-z]+$/.test(id))
                return true; // Solana address (base58)
            return false;
        };
        // Helper to detect Solana chain
        const isSolanaChain = (chain) => {
            if (!chain)
                return false;
            if (chain === 1399811149)
                return true;
            if (typeof chain === 'string') {
                const lower = chain.toLowerCase();
                return lower === 'solana' || lower === 'sol';
            }
            return false;
        };
        // Auto-fill user_id and wallet addresses
        let args = { ...rawArgs };
        if (args) {
            // Only auto-fill user_id for tools that need it
            const needsUserId = userIdTools.includes(tool);
            // For user_id tools, ALWAYS replace with real agent ID
            // Claude doesn't know the real ID so anything it sends is wrong
            if (needsUserId && agentId) {
                args.user_id = agentId;
                args.userId = agentId;
            }
            else {
                // For other tools, only replace "me" or "self"
                if (args.user_id === 'me' || args.user_id === 'self') {
                    if (agentId)
                        args.user_id = agentId;
                }
                if (args.userId === 'me' || args.userId === 'self') {
                    if (agentId)
                        args.userId = agentId;
                }
            }
            // Replace wallet placeholders with actual addresses
            // Supports: "my-wallet-evm", "my-wallet-svm", "me", "self"
            const replaceWalletPlaceholder = (value) => {
                if (value === 'my-wallet-evm' || value === 'me' || value === 'self') {
                    return evmAddress;
                }
                if (value === 'my-wallet-svm') {
                    return solanaAddress;
                }
                return undefined;
            };
            // Handle various wallet parameter names
            const walletParams = ['wallet', 'wallet_address', 'walletAddress', 'evm_address', 'taker', 'from_address', 'fromAddress'];
            for (const param of walletParams) {
                if (args[param]) {
                    const replacement = replaceWalletPlaceholder(args[param]);
                    if (replacement) {
                        args[param] = replacement;
                    }
                }
            }
            // Handle solana_address specifically (always use Solana wallet)
            if (args.solana_address === 'my-wallet-svm' || args.solana_address === 'me' || args.solana_address === 'self') {
                if (solanaAddress)
                    args.solana_address = solanaAddress;
            }
            // Auto-fill taker/from_address ONLY for swap execution tools (not read-only swap data tools)
            const swapExecutionTools = ['get_swap_price', 'get_swap_quote', 'execute_swap', 'execute_trade'];
            if (swapExecutionTools.includes(tool)) {
                const chain = args.src_chain || args.chain || args.chainId || args.network;
                // Fill taker if not set or placeholder
                if (!args.taker || args.taker === 'my-wallet-evm' || args.taker === 'my-wallet-svm') {
                    if (isSolanaChain(chain)) {
                        if (solanaAddress)
                            args.taker = solanaAddress;
                    }
                    else if (evmAddress) {
                        args.taker = evmAddress;
                    }
                }
                // Fill from_address if not set
                if (!args.from_address && !args.fromAddress) {
                    if (isSolanaChain(chain)) {
                        if (solanaAddress)
                            args.from_address = solanaAddress;
                    }
                    else if (evmAddress) {
                        args.from_address = evmAddress;
                    }
                }
            }
        }
        // Debug: log raw request from Claude and any modifications
        logger.claudeRequest(tool, rawArgs, args);
        // Create spinner for this tool call
        const spinner = createToolSpinner(tool, getToolDescription(tool));
        spinner.start();
        try {
            // Ensure we have a valid token
            let token = getAccessToken();
            if (!token || config.isTokenExpired()) {
                spinner.text = `${boba(tool)} ${bobaDim('Refreshing auth...')}`;
                const tokens = await refreshTokens();
                if (!tokens) {
                    spinner.fail(`${boba(tool)} ${chalk.hex('#FF6B6B')('Auth failed')}`);
                    res.status(401).json({ error: 'Authentication failed' });
                    return;
                }
                token = tokens.accessToken;
                spinner.text = `${boba(tool)} ${bobaDim(getToolDescription(tool))}`;
            }
            // Forward to MCP with auth and session info
            const mcpUrl = config.getMcpUrl();
            const response = await axios.post(`${mcpUrl}/call`, { tool, args }, {
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: `Bearer ${token}`,
                    // Pass session info to MCP (not in JWT but needed for wallet display)
                    'X-Agent-EVM-Address': evmAddress || '',
                    'X-Agent-Solana-Address': solanaAddress || '',
                    'X-Agent-Sub-Org-Id': tokens?.subOrganizationId || '',
                },
                timeout: 60000,
            });
            const duration = Date.now() - startTime;
            const preview = formatResult(tool, response.data);
            // Stop spinner with success
            spinner.succeed(`${boba(tool)} ${chalk.hex('#50FA7B')('✓')} ${bobaDim(`(${formatDuration(duration)})`)} ${preview ? bobaDim('→ ' + preview) : ''}`);
            // Show formatted output for supported tools
            const formatted = formatToolResult(tool, response.data);
            if (formatted) {
                console.log(formatted);
            }
            res.json(response.data);
        }
        catch (error) {
            const duration = Date.now() - startTime;
            if (axios.isAxiosError(error)) {
                const status = error.response?.status;
                const message = error.response?.data?.error || error.message;
                // Handle auth errors
                if (status === 401) {
                    spinner.text = `${boba(tool)} ${bobaDim('Re-authenticating...')}`;
                    const tokens = await authenticate();
                    if (tokens) {
                        // Retry the request
                        try {
                            spinner.text = `${boba(tool)} ${bobaDim('Retrying...')}`;
                            const mcpUrl = config.getMcpUrl();
                            const response = await axios.post(`${mcpUrl}/call`, { tool, args }, {
                                headers: {
                                    'Content-Type': 'application/json',
                                    Authorization: `Bearer ${tokens.accessToken}`,
                                    'X-Agent-EVM-Address': tokens.evmAddress || '',
                                    'X-Agent-Solana-Address': tokens.solanaAddress || '',
                                    'X-Agent-Sub-Org-Id': tokens.subOrganizationId || '',
                                },
                                timeout: 60000,
                            });
                            const retryDuration = Date.now() - startTime;
                            const preview = formatResult(tool, response.data);
                            spinner.succeed(`${boba(tool)} ${chalk.hex('#50FA7B')('✓')} ${bobaDim(`(${formatDuration(retryDuration)})`)} ${preview ? bobaDim('→ ' + preview) : ''}`);
                            // Show formatted output for supported tools
                            const formatted = formatToolResult(tool, response.data);
                            if (formatted) {
                                console.log(formatted);
                            }
                            res.json(response.data);
                            return;
                        }
                        catch (retryError) {
                            spinner.fail(`${boba(tool)} ${chalk.hex('#FF6B6B')('✗')} ${bobaDim(`(${formatDuration(Date.now() - startTime)})`)}`);
                            res.status(500).json({ error: retryError.message });
                            return;
                        }
                    }
                }
                spinner.fail(`${boba(tool)} ${chalk.hex('#FF6B6B')('✗')} ${bobaDim(message.slice(0, 50))}`);
                res.status(status || 500).json({ error: message });
            }
            else {
                spinner.fail(`${boba(tool)} ${chalk.hex('#FF6B6B')('✗')} ${bobaDim(error.message.slice(0, 50))}`);
                res.status(500).json({ error: error.message });
            }
        }
    });
    // SSE stream proxy
    app.get('/stream', async (req, res) => {
        res.setHeader('Content-Type', 'text/event-stream');
        res.setHeader('Cache-Control', 'no-cache');
        res.setHeader('Connection', 'keep-alive');
        const token = getAccessToken();
        if (!token) {
            res.write(`data: ${JSON.stringify({ error: 'Not authenticated' })}\n\n`);
            res.end();
            return;
        }
        try {
            const mcpUrl = config.getMcpUrl();
            const queryString = new URLSearchParams(req.query).toString();
            const response = await axios.get(`${mcpUrl}/stream?${queryString}`, {
                headers: { Authorization: `Bearer ${token}` },
                responseType: 'stream',
            });
            response.data.pipe(res);
            req.on('close', () => {
                response.data.destroy();
            });
        }
        catch (error) {
            res.write(`data: ${JSON.stringify({ error: error.message })}\n\n`);
            res.end();
        }
    });
    return {
        start: async () => {
            const port = config.getProxyPort();
            // Authenticate first
            const tokens = await ensureAuthenticated();
            if (!tokens) {
                throw new Error('Failed to authenticate');
            }
            return new Promise((resolve, reject) => {
                server = app.listen(port, '127.0.0.1', () => {
                    // Clear screen and show nice startup display
                    console.clear();
                    // Show the boba character (frame 2 - normal position)
                    console.log(colorizeFrame(bobaFrames[1]));
                    console.log();
                    // Centered status box - 44 char inner width
                    const boxW = 44;
                    const line = '═'.repeat(boxW);
                    const pad = (s, w) => s.length >= w ? s.slice(0, w) : s + ' '.repeat(w - s.length);
                    const center = (s, w) => {
                        const leftPad = Math.floor((w - s.length) / 2);
                        const rightPad = w - s.length - leftPad;
                        return ' '.repeat(leftPad) + s + ' '.repeat(rightPad);
                    };
                    console.log(matchaBright(`  ╔${line}╗`));
                    console.log(matchaBright('  ║') + matcha(center('🧋 BOBA PROXY ACTIVE', boxW)) + matchaBright('║'));
                    console.log(matchaBright(`  ╠${line}╣`));
                    console.log(matchaBright('  ║') + `  ${matcha('●')} Proxy:  ` + matchaBright(pad(`http://127.0.0.1:${port}`, 32)) + matchaBright('║'));
                    console.log(matchaBright('  ║') + `  ${matcha('●')} Agent:  ` + matchaBright(pad(tokens.agentName, 32)) + matchaBright('║'));
                    console.log(matchaBright('  ║') + `  ${matcha('●')} EVM:    ` + matchaDim(pad(tokens.evmAddress.slice(0, 32), 32)) + matchaBright('║'));
                    if (tokens.solanaAddress) {
                        console.log(matchaBright('  ║') + `  ${matcha('●')} Solana: ` + matchaDim(pad(tokens.solanaAddress.slice(0, 32), 32)) + matchaBright('║'));
                    }
                    console.log(matchaBright(`  ╠${line}╣`));
                    console.log(matchaBright('  ║') + matchaDim(center('Press Ctrl+C to stop', boxW)) + matchaBright('║'));
                    console.log(matchaBright(`  ╚${line}╝`));
                    console.log();
                    logger.section('ACTIVITY LOG');
                    // Block stdin - only allow Ctrl+C
                    if (process.stdin.isTTY) {
                        process.stdin.setRawMode(true);
                        process.stdin.resume();
                        process.stdin.on('data', (key) => {
                            // Ctrl+C
                            if (key[0] === 3) {
                                process.emit('SIGINT', 'SIGINT');
                            }
                            // Ignore all other input
                        });
                    }
                    resolve();
                });
                server.on('error', (err) => {
                    if (err.code === 'EADDRINUSE') {
                        logger.error(`Port ${port} is already in use. Try a different port with --port`);
                    }
                    else {
                        logger.error(`Failed to start server: ${err.message}`);
                    }
                    reject(err);
                });
            });
        },
        stop: () => {
            if (server) {
                server.close();
                logger.info('Proxy server stopped');
            }
        },
    };
}
