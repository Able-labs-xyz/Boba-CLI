import chalk, { type ChalkInstance } from 'chalk';

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
const toolColors: Record<string, ChalkInstance> = {
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

function getToolColor(tool: string): ChalkInstance {
  return toolColors[tool] || toolColors.default;
}

function timestamp(): string {
  const now = new Date();
  return colors.dim(`[${now.toLocaleTimeString()}]`);
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function truncate(str: string, maxLength: number = 50): string {
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength - 3) + '...';
}

export const logger = {
  // Standard logs
  info: (message: string) => {
    console.log(`${timestamp()} ${colors.info('ℹ')} ${message}`);
  },

  success: (message: string) => {
    console.log(`${timestamp()} ${colors.success('✓')} ${message}`);
  },

  warning: (message: string) => {
    console.log(`${timestamp()} ${colors.warning('⚠')} ${message}`);
  },

  error: (message: string) => {
    console.log(`${timestamp()} ${colors.error('✗')} ${colors.error(message)}`);
  },

  // Tool call logging
  toolCall: (tool: string, args: Record<string, any>) => {
    const color = getToolColor(tool);
    const argsStr = Object.entries(args)
      .map(([k, v]) => `${colors.dim(k)}=${colors.pearl(truncate(String(v)))}`)
      .join(' ');

    console.log(`${timestamp()} ${color('→')} ${color.bold(tool)} ${argsStr || colors.dim('(no args)')}`);
  },

  toolResult: (tool: string, success: boolean, duration: number, preview?: string) => {
    const color = getToolColor(tool);
    const icon = success ? colors.success('✓') : colors.error('✗');
    const durationStr = colors.dim(`(${formatDuration(duration)})`);
    const previewStr = preview ? ` ${colors.dim('→')} ${colors.pearl(truncate(preview, 40))}` : '';

    console.log(`${timestamp()} ${icon} ${color(tool)} ${durationStr}${previewStr}`);
  },

  // Request/Response logging
  request: (method: string, url: string) => {
    const methodColor = method === 'GET' ? colors.info : colors.warning;
    console.log(`${timestamp()} ${methodColor(method.padEnd(4))} ${colors.dim(url)}`);
  },

  response: (status: number, duration: number) => {
    const statusColor = status < 400 ? colors.success : colors.error;
    console.log(`${timestamp()} ${statusColor(String(status))} ${colors.dim(`(${formatDuration(duration)})`)}`);
  },

  // Connection status
  connected: (service: string) => {
    console.log(`${timestamp()} ${colors.success('●')} Connected to ${colors.matcha(service)}`);
  },

  disconnected: (service: string) => {
    console.log(`${timestamp()} ${colors.error('○')} Disconnected from ${colors.dim(service)}`);
  },

  // Proxy logs
  proxy: (direction: 'in' | 'out', tool: string, data?: any) => {
    const arrow = direction === 'in' ? colors.info('←') : colors.matcha('→');
    const preview = data ? ` ${colors.dim(truncate(JSON.stringify(data), 60))}` : '';
    console.log(`${timestamp()} ${arrow} ${colors.bold(tool)}${preview}`);
  },

  // Agent activity
  agentAction: (action: string, details?: string) => {
    console.log(`${timestamp()} ${colors.matcha('🤖')} ${colors.matchaBright(action)}${details ? ` ${colors.dim(details)}` : ''}`);
  },

  // Dividers and sections
  section: (title: string) => {
    const line = '─'.repeat(50);
    console.log(`\n${colors.matchaDim(line)}`);
    console.log(`  ${colors.matcha(title)}`);
    console.log(`${colors.matchaDim(line)}\n`);
  },

  // Raw data display (for debugging)
  data: (label: string, data: any) => {
    console.log(`${timestamp()} ${colors.dim(label + ':')} ${colors.pearl(JSON.stringify(data, null, 2))}`);
  },

  // Debug logging (shows raw request/response details)
  debug: (label: string, data: any) => {
    const debugEnabled = process.env.BOBA_DEBUG === '1' || process.env.BOBA_DEBUG === 'true';
    if (!debugEnabled) return;

    console.log(`${timestamp()} ${chalk.magenta('DEBUG')} ${chalk.magenta.bold(label)}`);
    console.log(chalk.gray(JSON.stringify(data, null, 2)));
  },

  // Claude request logging (always shows in debug mode)
  claudeRequest: (tool: string, rawArgs: any, modifiedArgs: any) => {
    const debugEnabled = process.env.BOBA_DEBUG === '1' || process.env.BOBA_DEBUG === 'true';
    if (!debugEnabled) return;

    console.log(`\n${timestamp()} ${chalk.cyan('┌──')} ${chalk.cyan.bold('CLAUDE REQUEST')}`);
    console.log(`${chalk.cyan('│')} Tool: ${chalk.yellow(tool)}`);
    console.log(`${chalk.cyan('│')} Raw args from Claude:`);
    console.log(chalk.gray(JSON.stringify(rawArgs, null, 2).split('\n').map(l => `${chalk.cyan('│')}   ${l}`).join('\n')));

    // Show if args were modified
    const rawStr = JSON.stringify(rawArgs);
    const modStr = JSON.stringify(modifiedArgs);
    if (rawStr !== modStr) {
      console.log(`${chalk.cyan('│')} ${chalk.yellow('⚡ Args modified by proxy:')}`);
      console.log(chalk.gray(JSON.stringify(modifiedArgs, null, 2).split('\n').map(l => `${chalk.cyan('│')}   ${l}`).join('\n')));
    }
    console.log(`${chalk.cyan('└──')}\n`);
  },

  // Blank line
  blank: () => console.log(),
};

export { colors };
