import { type AuthTokens } from './config.js';
export declare function authenticate(): Promise<AuthTokens | null>;
export declare function refreshTokens(): Promise<AuthTokens | null>;
export declare function ensureAuthenticated(): Promise<AuthTokens | null>;
export declare function getAccessToken(): string | null;
