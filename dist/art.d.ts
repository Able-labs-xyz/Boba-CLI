import ora, { type Ora } from 'ora';
declare const matcha: import("chalk").ChalkInstance;
declare const matchaDim: import("chalk").ChalkInstance;
declare const matchaBright: import("chalk").ChalkInstance;
declare const pearl: import("chalk").ChalkInstance;
declare const brown: import("chalk").ChalkInstance;
export declare const bobaGradient: (text: string) => string;
export declare const successGradient: (text: string) => string;
export declare const errorGradient: (text: string) => string;
declare const bobaFrames: string[][];
declare function colorizeFrame(frame: string[]): string;
export declare const BOBA_MINI: string;
export declare function printBanner(): void;
export declare function printStartup(): void;
export declare function bouncyLoader(message: string, duration?: number): Promise<void>;
export declare function bubbleLoader(message: string, duration?: number): Promise<void>;
export declare function spinnerLoader(message: string, style?: keyof typeof spinners): Promise<{
    stop: () => void;
    succeed: (text?: string) => void;
    fail: (text?: string) => void;
}>;
export declare const spinners: {
    dots: {
        interval: number;
        frames: string[];
    };
    line: {
        interval: number;
        frames: string[];
    };
    arc: {
        interval: number;
        frames: string[];
    };
    circle: {
        interval: number;
        frames: string[];
    };
    pulse: {
        interval: number;
        frames: string[];
    };
    bounce: {
        interval: number;
        frames: string[];
    };
    box: {
        interval: number;
        frames: string[];
    };
    arrows: {
        interval: number;
        frames: string[];
    };
    growing: {
        interval: number;
        frames: string[];
    };
};
export declare function createSpinner(text: string, style?: keyof typeof spinners): Ora;
export declare function withSpinner<T>(text: string, operation: () => Promise<T>, options?: {
    successText?: string;
    failText?: string;
}): Promise<T>;
export declare function gradientText(text: string): string;
export declare function successText(text: string): string;
export declare function errorText(text: string): string;
export declare function drawBox(title: string, content: string[], width?: number): string;
export declare function progressBar(current: number, total: number, width?: number): string;
export declare function sparkline(values: number[]): string;
export { matcha, matchaDim, matchaBright, pearl, brown, bobaFrames, colorizeFrame, ora };
