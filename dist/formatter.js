import asciichart from 'asciichart';
import Table from 'cli-table3';
import chalk from 'chalk';
import gradient from 'gradient-string';
// Boba purple theme
const boba = chalk.hex('#B184F5');
const bobaBright = chalk.hex('#D4A5FF');
const bobaDim = chalk.hex('#8A5FD1');
const gold = chalk.hex('#FFD700');
const red = chalk.hex('#FF6B6B');
const cyan = chalk.hex('#00CED1');
const green = chalk.hex('#50FA7B');
// Legacy aliases for compatibility
const matcha = boba;
const matchaBright = bobaBright;
const matchaDim = bobaDim;
// Gradients
const bobaGradient = gradient(['#8A5FD1', '#B184F5', '#D4A5FF']);
const goldGradient = gradient(['#FFD700', '#FFA500']);
const profitGradient = gradient(['#50FA7B', '#69FF94']);
const lossGradient = gradient(['#FF6B6B', '#FF4444']);
// Sparkline mini chart
function sparkline(values) {
    if (!values || values.length === 0)
        return '';
    const chars = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];
    const min = Math.min(...values);
    const max = Math.max(...values);
    const range = max - min || 1;
    return values.map(v => {
        const index = Math.floor(((v - min) / range) * (chars.length - 1));
        return boba(chars[index]);
    }).join('');
}
// Progress bar
function progressBar(current, total, width = 20) {
    const percentage = Math.min(100, Math.max(0, (current / total) * 100));
    const filled = Math.round((percentage / 100) * width);
    const empty = width - filled;
    return boba('█'.repeat(filled)) + bobaDim('░'.repeat(empty));
}
// Format currency
function formatUSD(value) {
    if (value === undefined || value === null)
        return matchaDim('--');
    const num = typeof value === 'string' ? parseFloat(value) : value;
    if (isNaN(num))
        return matchaDim('--');
    if (num >= 1_000_000_000)
        return gold(`$${(num / 1_000_000_000).toFixed(2)}B`);
    if (num >= 1_000_000)
        return gold(`$${(num / 1_000_000).toFixed(2)}M`);
    if (num >= 1_000)
        return gold(`$${(num / 1_000).toFixed(2)}K`);
    if (num >= 1)
        return gold(`$${num.toFixed(2)}`);
    return gold(`$${num.toFixed(6)}`);
}
// Format percentage with color
function formatPercent(value) {
    if (value === undefined || value === null)
        return matchaDim('--');
    const num = typeof value === 'string' ? parseFloat(value) : value;
    if (isNaN(num))
        return matchaDim('--');
    const color = num >= 0 ? matcha : red;
    const sign = num >= 0 ? '+' : '';
    return color(`${sign}${num.toFixed(2)}%`);
}
// Format large numbers
function formatNumber(value) {
    if (value === undefined || value === null)
        return matchaDim('--');
    const num = typeof value === 'string' ? parseFloat(value) : value;
    if (isNaN(num))
        return matchaDim('--');
    if (num >= 1_000_000_000)
        return `${(num / 1_000_000_000).toFixed(2)}B`;
    if (num >= 1_000_000)
        return `${(num / 1_000_000).toFixed(2)}M`;
    if (num >= 1_000)
        return `${(num / 1_000).toFixed(2)}K`;
    return num.toFixed(2);
}
// Truncate address
function truncateAddress(addr) {
    if (!addr)
        return matchaDim('--');
    return matchaDim(`${addr.slice(0, 6)}...${addr.slice(-4)}`);
}
// Format portfolio response
export function formatPortfolio(data) {
    const lines = [];
    const totalValue = data.total_value_usd || data.totalValueUsd || 0;
    const pnl = data.pnl_24h || data.pnl24h || 0;
    const pnlPositive = pnl >= 0;
    lines.push('');
    lines.push(bobaBright('  ╔═════════════════════════════════════════════════════════════╗'));
    lines.push(bobaBright('  ║') + bobaGradient('                      💎 PORTFOLIO 💎                       ') + bobaBright('║'));
    lines.push(bobaBright('  ╠═════════════════════════════════════════════════════════════╣'));
    // Total value with big display
    const valueDisplay = goldGradient(`  $${Number(totalValue).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`);
    lines.push(bobaBright('  ║') + valueDisplay.padEnd(72) + bobaBright('║'));
    // PnL with color-coded gradient
    if (pnl !== 0) {
        const pnlStr = pnlPositive ? `  ▲ +${pnl.toFixed(2)}%` : `  ▼ ${pnl.toFixed(2)}%`;
        const pnlColored = pnlPositive ? profitGradient(pnlStr) : lossGradient(pnlStr);
        lines.push(bobaBright('  ║') + pnlColored.padEnd(72) + bobaBright('║'));
    }
    lines.push(bobaBright('  ╠═════════════════════════════════════════════════════════════╣'));
    // Token holdings with mini charts
    const tokens = data.tokens || data.holdings || [];
    if (tokens.length > 0) {
        lines.push(bobaBright('  ║') + bobaGradient('  HOLDINGS                                                   ') + bobaBright('║'));
        lines.push(bobaBright('  ║') + bobaDim('  ───────────────────────────────────────────────────────────') + bobaBright('║'));
        tokens.slice(0, 8).forEach((token) => {
            const symbol = (token.symbol || token.token?.symbol || '???').padEnd(8);
            const value = formatUSD(token.value_usd || token.valueUsd || token.value);
            const balance = formatNumber(token.balance || token.amount);
            const tokenPnl = token.pnl_24h || token.change24h || 0;
            // Calculate allocation percentage
            const allocation = totalValue > 0 ? ((token.value_usd || token.valueUsd || 0) / totalValue) * 100 : 0;
            const allocBar = progressBar(allocation, 100, 10);
            // Token row with allocation bar
            const pnlIndicator = tokenPnl >= 0 ? green('▲') : red('▼');
            lines.push(bobaBright('  ║') + `  ${boba(symbol)} ${value.padEnd(12)} ${allocBar} ${bobaDim(balance.padEnd(10))} ${pnlIndicator}`.padEnd(61) + bobaBright('║'));
        });
        if (tokens.length > 8) {
            lines.push(bobaBright('  ║') + bobaDim(`  ... and ${tokens.length - 8} more tokens`).padEnd(61) + bobaBright('║'));
        }
    }
    lines.push(bobaBright('  ╚═════════════════════════════════════════════════════════════╝'));
    lines.push('');
    return lines.join('\n');
}
// Format token search results
export function formatTokenSearch(data) {
    const tokens = data.tokens || data.results || data || [];
    if (!Array.isArray(tokens) || tokens.length === 0) {
        return matchaDim('\n  No tokens found\n');
    }
    const table = new Table({
        head: [
            chalk.white('Symbol'),
            chalk.white('Price'),
            chalk.white('MCap'),
            chalk.white('Vol 24h'),
            chalk.white('Liq'),
        ],
        style: {
            head: [],
            border: ['grey'],
        },
        colWidths: [12, 14, 12, 12, 12],
    });
    tokens.slice(0, 15).forEach((t) => {
        table.push([
            matcha(t.symbol || '???'),
            formatUSD(t.price_usd || t.priceUsd || t.price),
            formatNumber(t.market_cap || t.marketCap),
            formatNumber(t.volume_24h || t.volume24h || t.volume),
            formatNumber(t.liquidity),
        ]);
    });
    return '\n' + table.toString() + '\n';
}
// Format token info
export function formatTokenInfo(data) {
    const lines = [];
    lines.push('');
    lines.push(matchaBright(`  ══════════════════════════════════════════════════════`));
    lines.push(matcha(`  ${data.name || 'Unknown'} (${data.symbol || '???'})`));
    lines.push(matchaBright(`  ══════════════════════════════════════════════════════`));
    lines.push('');
    lines.push(`  ${matchaDim('Price:')}        ${formatUSD(data.price_usd || data.price)}`);
    lines.push(`  ${matchaDim('Market Cap:')}  ${formatUSD(data.market_cap || data.marketCap)}`);
    lines.push(`  ${matchaDim('Volume 24h:')} ${formatUSD(data.volume_24h || data.volume)}`);
    lines.push(`  ${matchaDim('Liquidity:')}   ${formatUSD(data.liquidity)}`);
    lines.push(`  ${matchaDim('Holders:')}     ${formatNumber(data.holders)}`);
    lines.push(`  ${matchaDim('Address:')}     ${truncateAddress(data.address)}`);
    // Token audit / security info
    if (data.audit || data.security) {
        const audit = data.audit || data.security || {};
        lines.push('');
        lines.push(cyan(`  Security Audit:`));
        const checkMark = (val) => val ? matcha('✓') : red('✗');
        if (audit.is_honeypot !== undefined) {
            lines.push(`    ${audit.is_honeypot ? red('⚠ HONEYPOT') : matcha('✓ Not Honeypot')}`);
        }
        if (audit.is_mintable !== undefined) {
            lines.push(`    ${matchaDim('Mintable:')}     ${checkMark(!audit.is_mintable)}`);
        }
        if (audit.can_blacklist !== undefined) {
            lines.push(`    ${matchaDim('Blacklist:')}    ${checkMark(!audit.can_blacklist)}`);
        }
        if (audit.buy_tax !== undefined) {
            lines.push(`    ${matchaDim('Buy Tax:')}      ${audit.buy_tax}%`);
        }
        if (audit.sell_tax !== undefined) {
            lines.push(`    ${matchaDim('Sell Tax:')}     ${audit.sell_tax}%`);
        }
    }
    lines.push('');
    return lines.join('\n');
}
// Format PnL chart
export function formatPnLChart(data) {
    const lines = [];
    const chartData = data.chart || data.data || data.points || [];
    if (!Array.isArray(chartData) || chartData.length < 2) {
        return matchaDim('\n  Not enough data for chart\n');
    }
    // Extract values for chart
    const values = chartData.map((p) => {
        const val = p.value || p.pnl || p.total_value_usd || p.y || 0;
        return typeof val === 'string' ? parseFloat(val) : val;
    }).filter((v) => !isNaN(v));
    if (values.length < 2) {
        return matchaDim('\n  Not enough data for chart\n');
    }
    lines.push('');
    lines.push(matchaBright('  Portfolio Value Over Time'));
    lines.push(matchaDim('  ─────────────────────────────────────────────────────'));
    try {
        const chart = asciichart.plot(values, {
            height: 10,
            colors: [asciichart.magenta],
            format: (x) => formatUSD(x).padStart(10),
        });
        lines.push(chart);
        // Add sparkline summary
        const sparkValues = values.slice(-30); // Last 30 points
        lines.push(`  ${bobaDim('Trend:')} ${sparkline(sparkValues)}`);
    }
    catch (e) {
        lines.push(bobaDim('  Could not render chart'));
    }
    lines.push(matchaDim('  ─────────────────────────────────────────────────────'));
    // Summary stats
    const first = values[0];
    const last = values[values.length - 1];
    const change = ((last - first) / first) * 100;
    const max = Math.max(...values);
    const min = Math.min(...values);
    lines.push(`  ${matchaDim('Start:')} ${formatUSD(first)}  ${matchaDim('End:')} ${formatUSD(last)}  ${matchaDim('Change:')} ${formatPercent(change)}`);
    lines.push(`  ${matchaDim('High:')}  ${formatUSD(max)}  ${matchaDim('Low:')}  ${formatUSD(min)}`);
    lines.push('');
    return lines.join('\n');
}
// Format swap quote
export function formatSwapQuote(data) {
    const lines = [];
    lines.push('');
    lines.push(matchaBright('  ╔═══════════════════════════════════════════════════════╗'));
    lines.push(matchaBright('  ║') + cyan('                    SWAP QUOTE                         ') + matchaBright('║'));
    lines.push(matchaBright('  ╠═══════════════════════════════════════════════════════╣'));
    const fromAmount = data.src_amount || data.fromAmount || data.inputAmount;
    const toAmount = data.dst_amount || data.toAmount || data.outputAmount;
    const fromSymbol = data.src_token?.symbol || data.fromToken?.symbol || 'TOKEN';
    const toSymbol = data.dst_token?.symbol || data.toToken?.symbol || 'TOKEN';
    lines.push(matchaBright('  ║') + `  ${matchaDim('From:')}   ${matcha(formatNumber(fromAmount))} ${fromSymbol}`.padEnd(52) + matchaBright('║'));
    lines.push(matchaBright('  ║') + `  ${matchaDim('To:')}     ${gold(formatNumber(toAmount))} ${toSymbol}`.padEnd(52) + matchaBright('║'));
    if (data.price_impact !== undefined || data.priceImpact !== undefined) {
        const impact = data.price_impact || data.priceImpact;
        lines.push(matchaBright('  ║') + `  ${matchaDim('Impact:')} ${formatPercent(-Math.abs(impact))}`.padEnd(52) + matchaBright('║'));
    }
    if (data.gas_estimate || data.gasEstimate) {
        lines.push(matchaBright('  ║') + `  ${matchaDim('Gas:')}    ~${formatUSD(data.gas_estimate || data.gasEstimate)}`.padEnd(52) + matchaBright('║'));
    }
    lines.push(matchaBright('  ╚═══════════════════════════════════════════════════════╝'));
    lines.push('');
    return lines.join('\n');
}
// Format trade execution result
export function formatTradeResult(data) {
    const lines = [];
    const success = data.success || data.status === 'success' || data.tx_hash;
    lines.push('');
    if (success) {
        lines.push(boba('  ╔═══════════════════════════════════════════════════════╗'));
        lines.push(boba('  ║') + profitGradient('           ✨ TRADE EXECUTED SUCCESSFULLY ✨           ') + boba('║'));
        lines.push(boba('  ╠═══════════════════════════════════════════════════════╣'));
        if (data.tx_hash || data.txHash) {
            const hash = data.tx_hash || data.txHash;
            lines.push(boba('  ║') + `  ${bobaDim('TX:')} ${boba(hash.slice(0, 20))}...${boba(hash.slice(-8))}`.padEnd(52) + boba('║'));
        }
        if (data.amount_in && data.amount_out) {
            lines.push(boba('  ║') + `  ${bobaDim('Swapped:')} ${gold(formatNumber(data.amount_in))} → ${green(formatNumber(data.amount_out))}`.padEnd(52) + boba('║'));
        }
        if (data.token_in && data.token_out) {
            lines.push(boba('  ║') + `  ${bobaDim('Tokens:')} ${data.token_in.slice(0, 8)}... → ${data.token_out.slice(0, 8)}...`.padEnd(52) + boba('║'));
        }
        lines.push(boba('  ╚═══════════════════════════════════════════════════════╝'));
    }
    else {
        lines.push(red('  ╔═══════════════════════════════════════════════════════╗'));
        lines.push(red('  ║') + lossGradient('              ✗ TRADE FAILED                          ') + red('║'));
        lines.push(red('  ╠═══════════════════════════════════════════════════════╣'));
        if (data.error || data.message) {
            const errorMsg = (data.error || data.message).slice(0, 48);
            lines.push(red('  ║') + `  ${red(errorMsg)}`.padEnd(55) + red('║'));
        }
        lines.push(red('  ╚═══════════════════════════════════════════════════════╝'));
    }
    lines.push('');
    return lines.join('\n');
}
// Format trending tokens
export function formatTrendingTokens(data) {
    const tokens = data.tokens || data.trending || data || [];
    if (!Array.isArray(tokens) || tokens.length === 0) {
        return bobaDim('\n  No trending tokens\n');
    }
    const fireGradient = gradient(['#FF6B6B', '#FFD93D', '#FF6B6B']);
    const lines = [];
    lines.push('');
    lines.push(bobaBright('  ╔═════════════════════════════════════════════════════════════╗'));
    lines.push(bobaBright('  ║') + fireGradient('                 🔥 TRENDING TOKENS 🔥                      ') + bobaBright('║'));
    lines.push(bobaBright('  ╠═════════════════════════════════════════════════════════════╣'));
    tokens.slice(0, 10).forEach((t, i) => {
        const medal = i === 0 ? '🥇' : i === 1 ? '🥈' : i === 2 ? '🥉' : `${(i + 1).toString().padStart(2)}.`;
        const symbol = boba((t.symbol || '???').padEnd(10));
        const price = formatUSD(t.price_usd || t.price).padEnd(14);
        const change = t.price_change_24h || t.change24h || 0;
        const changeStr = change >= 0
            ? green(`▲ +${Number(change).toFixed(1)}%`)
            : red(`▼ ${Number(change).toFixed(1)}%`);
        lines.push(bobaBright('  ║') + `  ${medal} ${symbol} ${price} ${changeStr}`.padEnd(61) + bobaBright('║'));
    });
    lines.push(bobaBright('  ╚═════════════════════════════════════════════════════════════╝'));
    lines.push('');
    return lines.join('\n');
}
// Format token price chart (OHLC data)
export function formatTokenChart(data) {
    const lines = [];
    // Handle various data formats
    let chartData = data.candles || data.ohlc || data.data || data.chart || data.bars || data;
    if (!Array.isArray(chartData)) {
        // Try to find array in nested structure
        const keys = Object.keys(data);
        for (const key of keys) {
            if (Array.isArray(data[key])) {
                chartData = data[key];
                break;
            }
        }
    }
    if (!Array.isArray(chartData) || chartData.length < 2) {
        return matchaDim('\n  Not enough data for chart\n');
    }
    // Extract close prices for the chart
    const values = chartData.map((candle) => {
        // Handle different OHLC formats
        const price = candle.close || candle.c || candle.price || candle.value || candle[4] || candle;
        return typeof price === 'string' ? parseFloat(price) : price;
    }).filter((v) => !isNaN(v) && v > 0);
    if (values.length < 2) {
        return matchaDim('\n  Not enough price data for chart\n');
    }
    // Get token info if available
    const symbol = data.symbol || data.token?.symbol || '';
    const timeframe = data.timeframe || data.interval || '';
    lines.push('');
    lines.push(matchaBright(`  📈 ${symbol ? symbol + ' ' : ''}PRICE CHART${timeframe ? ' (' + timeframe + ')' : ''}`));
    lines.push(matchaDim('  ─────────────────────────────────────────────────────'));
    try {
        const chart = asciichart.plot(values, {
            height: 12,
            colors: [asciichart.magenta],
            format: (x) => {
                if (x >= 1)
                    return ('$' + x.toFixed(2)).padStart(12);
                if (x >= 0.01)
                    return ('$' + x.toFixed(4)).padStart(12);
                return ('$' + x.toFixed(8)).padStart(12);
            },
        });
        lines.push(chart);
        // Add sparkline summary
        const sparkValues = values.slice(-30);
        lines.push(`  ${bobaDim('Mini:')} ${sparkline(sparkValues)}`);
    }
    catch (e) {
        lines.push(bobaDim('  Could not render chart'));
    }
    lines.push(matchaDim('  ─────────────────────────────────────────────────────'));
    // Summary stats
    const first = values[0];
    const last = values[values.length - 1];
    const change = ((last - first) / first) * 100;
    const max = Math.max(...values);
    const min = Math.min(...values);
    lines.push(`  ${matchaDim('Open:')}  ${formatUSD(first)}   ${matchaDim('Close:')} ${formatUSD(last)}   ${matchaDim('Change:')} ${formatPercent(change)}`);
    lines.push(`  ${matchaDim('High:')}  ${formatUSD(max)}   ${matchaDim('Low:')}   ${formatUSD(min)}`);
    lines.push('');
    return lines.join('\n');
}
// Main formatter - detect type and format accordingly
export function formatToolResult(tool, data) {
    try {
        // Parse if string
        const parsed = typeof data === 'string' ? JSON.parse(data) : data;
        switch (tool) {
            case 'get_portfolio':
                return formatPortfolio(parsed);
            case 'search_tokens':
            case 'get_tokens_by_category':
                return formatTokenSearch(parsed);
            case 'get_token_info':
            case 'get_token_details':
                return formatTokenInfo(parsed);
            case 'get_portfolio_pnl':
            case 'get_pnl_chart':
                return formatPnLChart(parsed);
            case 'get_token_chart':
            case 'get_token_ohlc':
            case 'get_ohlc':
                return formatTokenChart(parsed);
            case 'get_swap_price':
            case 'get_swap_quote':
                return formatSwapQuote(parsed);
            case 'execute_swap':
            case 'execute_trade':
                return formatTradeResult(parsed);
            case 'get_trending_tokens':
                return formatTrendingTokens(parsed);
            default:
                return null; // No special formatting
        }
    }
    catch (e) {
        return null; // Fall back to default
    }
}
