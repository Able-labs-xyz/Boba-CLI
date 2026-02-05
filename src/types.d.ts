declare module 'asciichart' {
  export function plot(series: number[], config?: {
    height?: number;
    offset?: number;
    padding?: string;
    colors?: number[];
    format?: (x: number) => string;
  }): string;

  export const green: number;
  export const red: number;
  export const yellow: number;
  export const blue: number;
  export const cyan: number;
  export const magenta: number;
  export const white: number;
  export const black: number;
}
