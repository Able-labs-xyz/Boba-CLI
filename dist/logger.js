import chalk from 'chalk';
// Color scheme matching Boba brand - purple theme
const colors = {
    matcha: chalk.hex('#B184F5'),
    matchaDim: chalk.hex('#8A5FD1'),
    matchaBright: chalk.hex('#D4A5FF'),
    pearl: chalk.hex('#F5F5DC'),
    brown: chalk.hex('#8B4513'),
    error: chalk.hex('#FF6B6B'),
    warning: chalk.hex('#FFE66D'),
    info: chalk.hex('#4ECDC4'),
    success: chalk.hex('#B184F5'),
    dim: chalk.dim,
    bold: chalk.bold,
};
// Tool category colors
const toolColors = {
    // Trading
    get_swap_price: chalk.hex('#4ECDC4'),
    execute_swap: chalk.hex('#FF6B6B'),
    get_swap_quote: chalk.hex('#4ECDC4'),
    // Portfolio
    get_portfolio: chalk.hex('#9B59B6'),
    get_portfolio_pnl: chalk.hex('#9B59B6'),
    get_portfolio_history: chalk.hex('#9B59B6'),
    // Token info
    get_token_info: chalk.hex('#F39C12'),
    search_tokens: chalk.hex('#F39C12'),
    get_trending_tokens: chalk.hex('#F39C12'),
    // Wallet
    get_wallet_balance: chalk.hex('#3498DB'),
    get_wallet_transactions: chalk.hex('#3498DB'),
    // Brewing (new launches)
    get_brewing_status: chalk.hex('#E74C3C'),
    get_recent_launches: chalk.hex('#E74C3C'),
    stream_launches: chalk.hex('#E74C3C'),
    // Default
    default: chalk.hex('#B184F5'),
};
function getToolColor(tool) {
    return toolColors[tool] || toolColors.default;
}
function timestamp() {
    const now = new Date();
    return colors.dim(`[${now.toLocaleTimeString()}]`);
}
function formatDuration(ms) {
    if (ms < 1000)
        return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
}
function truncate(str, maxLength = 50) {
    if (str.length <= maxLength)
        return str;
    return str.substring(0, maxLength - 3) + '...';
}
export const logger = {
    // Standard logs
    info: (message) => {
        console.log(`${timestamp()} ${colors.info('ℹ')} ${message}`);
    },
    success: (message) => {
        console.log(`${timestamp()} ${colors.success('✓')} ${message}`);
    },
    warning: (message) => {
        console.log(`${timestamp()} ${colors.warning('⚠')} ${message}`);
    },
    error: (message) => {
        console.log(`${timestamp()} ${colors.error('✗')} ${colors.error(message)}`);
    },
    // Tool call logging
    toolCall: (tool, args) => {
        const color = getToolColor(tool);
        const argsStr = Object.entries(args)
            .map(([k, v]) => `${colors.dim(k)}=${colors.pearl(truncate(String(v)))}`)
            .join(' ');
        console.log(`${timestamp()} ${color('→')} ${color.bold(tool)} ${argsStr || colors.dim('(no args)')}`);
    },
    toolResult: (tool, success, duration, preview) => {
        const color = getToolColor(tool);
        const icon = success ? colors.success('✓') : colors.error('✗');
        const durationStr = colors.dim(`(${formatDuration(duration)})`);
        const previewStr = preview ? ` ${colors.dim('→')} ${colors.pearl(truncate(preview, 40))}` : '';
        console.log(`${timestamp()} ${icon} ${color(tool)} ${durationStr}${previewStr}`);
    },
    // Request/Response logging
    request: (method, url) => {
        const methodColor = method === 'GET' ? colors.info : colors.warning;
        console.log(`${timestamp()} ${methodColor(method.padEnd(4))} ${colors.dim(url)}`);
    },
    response: (status, duration) => {
        const statusColor = status < 400 ? colors.success : colors.error;
        console.log(`${timestamp()} ${statusColor(String(status))} ${colors.dim(`(${formatDuration(duration)})`)}`);
    },
    // Connection status
    connected: (service) => {
        console.log(`${timestamp()} ${colors.success('●')} Connected to ${colors.matcha(service)}`);
    },
    disconnected: (service) => {
        console.log(`${timestamp()} ${colors.error('○')} Disconnected from ${colors.dim(service)}`);
    },
    // Proxy logs
    proxy: (direction, tool, data) => {
        const arrow = direction === 'in' ? colors.info('←') : colors.matcha('→');
        const preview = data ? ` ${colors.dim(truncate(JSON.stringify(data), 60))}` : '';
        console.log(`${timestamp()} ${arrow} ${colors.bold(tool)}${preview}`);
    },
    // Agent activity
    agentAction: (action, details) => {
        console.log(`${timestamp()} ${colors.matcha('🤖')} ${colors.matchaBright(action)}${details ? ` ${colors.dim(details)}` : ''}`);
    },
    // Dividers and sections
    section: (title) => {
        const line = '─'.repeat(50);
        console.log(`\n${colors.matchaDim(line)}`);
        console.log(`  ${colors.matcha(title)}`);
        console.log(`${colors.matchaDim(line)}\n`);
    },
    // Blank line
    blank: () => console.log(),
};
export { colors };
