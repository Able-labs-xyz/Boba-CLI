import chalk from 'chalk';
import ora from 'ora';
import gradient from 'gradient-string';
// Boba purple color theme
const matcha = chalk.hex('#B184F5');
const matchaDim = chalk.hex('#8A5FD1');
const matchaBright = chalk.hex('#D4A5FF');
const pearl = chalk.hex('#F5F5DC');
const brown = chalk.hex('#8B4513');
// Custom gradient for boba (internal use)
const bobaGrad = gradient(['#8A5FD1', '#B184F5', '#D4A5FF']);
const successGrad = gradient(['#50FA7B', '#69FF94']);
const errorGrad = gradient(['#FF6B6B', '#FF8E8E']);
// Exported gradient text functions
export const bobaGradient = (text) => bobaGrad(text);
export const successGradient = (text) => successGrad(text);
export const errorGradient = (text) => errorGrad(text);
// BOBA AGENTS ASCII logo (matches agents-web frontend)
const BOBA_LOGO = [
    `        ██████╗  ██████╗ ██████╗  █████╗        `,
    `        ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗       `,
    `        ██████╔╝██║   ██║██████╔╝███████║       `,
    `        ██╔══██╗██║   ██║██╔══██╗██╔══██║       `,
    `        ██████╔╝╚██████╔╝██████╔╝██║  ██║       `,
    `        ╚═════╝  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝       `,
    `                             A G E N T S       `,
];
// Legacy bobaFrames for backward compatibility (uses logo)
const bobaFrames = [BOBA_LOGO, BOBA_LOGO, BOBA_LOGO, BOBA_LOGO];
// Colorize the BOBA logo
function colorizeLogo() {
    return BOBA_LOGO.map(line => matcha(line)).join('\n');
}
// Colorize a frame (for backward compatibility)
function colorizeFrame(frame) {
    return frame.map(line => matcha(line)).join('\n');
}
export const BOBA_MINI = `${matcha('🧋 boba')}`;
// Print the BOBA AGENTS banner
export function printBanner() {
    console.log();
    console.log(colorizeLogo());
    console.log();
}
export function printStartup() {
    console.clear();
    printBanner();
}
// Animated loading with boba character + spinner
export async function bouncyLoader(message, duration = 3000) {
    const startTime = Date.now();
    let frameIndex = 0;
    let spinnerIndex = 0;
    const frameDelay = 200;
    const spinnerFrames = spinners.dots.frames;
    process.stdout.write('\x1B[?25l');
    return new Promise((resolve) => {
        const interval = setInterval(() => {
            const elapsed = Date.now() - startTime;
            if (elapsed >= duration) {
                clearInterval(interval);
                process.stdout.write('\x1B[?25h');
                process.stdout.write('\x1B[2J\x1B[H');
                resolve();
                return;
            }
            process.stdout.write('\x1B[2J\x1B[H');
            console.log(colorizeFrame(bobaFrames[frameIndex]));
            console.log();
            const spinner = matcha(spinnerFrames[spinnerIndex % spinnerFrames.length]);
            const dots = '.'.repeat((Math.floor(elapsed / 500) % 4));
            console.log(`             ${spinner} ${matchaDim(message)}${matchaDim(dots)}`);
            frameIndex = (frameIndex + 1) % bobaFrames.length;
            spinnerIndex++;
        }, frameDelay);
    });
}
// Inline spinner loader (no full screen clear)
export async function bubbleLoader(message, duration = 1500) {
    const spinner = createSpinner(message, 'dots');
    spinner.start();
    return new Promise((resolve) => {
        setTimeout(() => {
            spinner.stop();
            process.stdout.write('\r' + ' '.repeat(60) + '\r');
            resolve();
        }, duration);
    });
}
// Ora-based spinner that can be controlled
export async function spinnerLoader(message, style = 'dots') {
    const spinner = createSpinner(message, style);
    spinner.start();
    return {
        stop: () => spinner.stop(),
        succeed: (text) => spinner.succeed(text ? matcha(text) : undefined),
        fail: (text) => spinner.fail(text ? chalk.hex('#FF6B6B')(text) : undefined),
    };
}
// Animated spinner styles (no emojis, pure CLI aesthetic)
export const spinners = {
    dots: {
        interval: 80,
        frames: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
    },
    line: {
        interval: 130,
        frames: ['—', '\\', '|', '/'],
    },
    arc: {
        interval: 100,
        frames: ['◜', '◠', '◝', '◞', '◡', '◟'],
    },
    circle: {
        interval: 120,
        frames: ['◐', '◓', '◑', '◒'],
    },
    pulse: {
        interval: 100,
        frames: ['█', '▓', '▒', '░', '▒', '▓'],
    },
    bounce: {
        interval: 120,
        frames: ['⠁', '⠂', '⠄', '⡀', '⢀', '⠠', '⠐', '⠈'],
    },
    box: {
        interval: 120,
        frames: ['▖', '▘', '▝', '▗'],
    },
    arrows: {
        interval: 100,
        frames: ['←', '↖', '↑', '↗', '→', '↘', '↓', '↙'],
    },
    growing: {
        interval: 120,
        frames: ['▁', '▃', '▄', '▅', '▆', '▇', '█', '▇', '▆', '▅', '▄', '▃'],
    },
};
// Create ora spinner with boba purple theme
export function createSpinner(text, style = 'dots') {
    const spinner = spinners[style];
    return ora({
        text: matcha(text),
        spinner: {
            interval: spinner.interval,
            frames: spinner.frames.map(f => matcha(f)),
        },
        color: 'magenta',
    });
}
// Fancy progress display for operations
export async function withSpinner(text, operation, options) {
    const spinner = createSpinner(text);
    spinner.start();
    try {
        const result = await operation();
        spinner.succeed(options?.successText ? matcha(options.successText) : matcha('Done!'));
        return result;
    }
    catch (error) {
        spinner.fail(options?.failText ? chalk.hex('#FF6B6B')(options.failText) : chalk.hex('#FF6B6B')('Failed'));
        throw error;
    }
}
// Gradient text helpers
export function gradientText(text) {
    return bobaGrad(text);
}
export function successText(text) {
    return successGrad(text);
}
export function errorText(text) {
    return errorGrad(text);
}
// Fancy box drawing
export function drawBox(title, content, width = 50) {
    const lines = [];
    const innerWidth = width - 4;
    // Top border with gradient
    lines.push(matchaBright('  ╔' + '═'.repeat(innerWidth) + '╗'));
    // Title
    const titlePadded = title.padStart(Math.floor((innerWidth + title.length) / 2)).padEnd(innerWidth);
    lines.push(matchaBright('  ║') + bobaGradient(titlePadded) + matchaBright('║'));
    // Separator
    lines.push(matchaBright('  ╠' + '═'.repeat(innerWidth) + '╣'));
    // Content
    for (const line of content) {
        const paddedLine = line.slice(0, innerWidth).padEnd(innerWidth);
        lines.push(matchaBright('  ║') + paddedLine + matchaBright('║'));
    }
    // Bottom border
    lines.push(matchaBright('  ╚' + '═'.repeat(innerWidth) + '╝'));
    return lines.join('\n');
}
// Progress bar
export function progressBar(current, total, width = 30) {
    const percentage = Math.min(100, Math.max(0, (current / total) * 100));
    const filled = Math.round((percentage / 100) * width);
    const empty = width - filled;
    const filledBar = matcha('█'.repeat(filled));
    const emptyBar = matchaDim('░'.repeat(empty));
    return `${filledBar}${emptyBar} ${matchaBright(percentage.toFixed(0) + '%')}`;
}
// Sparkline mini chart
export function sparkline(values) {
    const chars = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];
    const min = Math.min(...values);
    const max = Math.max(...values);
    const range = max - min || 1;
    return values.map(v => {
        const index = Math.floor(((v - min) / range) * (chars.length - 1));
        return matcha(chars[index]);
    }).join('');
}
export { matcha, matchaDim, matchaBright, pearl, brown, bobaFrames, colorizeFrame, ora };
