interface AgentCredentials {
    agentId: string;
    agentSecret: string;
    name?: string;
}
interface AuthTokens {
    accessToken: string;
    refreshToken: string;
    accessTokenExpiresAt: string;
    refreshTokenExpiresAt: string;
    agentId: string;
    agentName: string;
    evmAddress: string;
    solanaAddress: string;
    subOrganizationId: string;
}
interface BobaConfig {
    credentials?: AgentCredentials;
    tokens?: AuthTokens;
    mcpUrl: string;
    authUrl: string;
    proxyPort: number;
    logLevel: 'debug' | 'info' | 'warn' | 'error';
}
export declare const config: {
    getCredentials: () => Promise<AgentCredentials | undefined>;
    setCredentials: (creds: AgentCredentials) => Promise<void>;
    clearCredentials: () => Promise<void>;
    hasCredentials: () => boolean;
    getTokens: () => Promise<AuthTokens | undefined>;
    getTokensSync: () => AuthTokens | undefined;
    setTokens: (tokens: AuthTokens) => Promise<void>;
    clearTokens: () => Promise<void>;
    isTokenExpired: () => boolean;
    getMcpUrl: () => string;
    setMcpUrl: (url: string, force?: boolean) => void;
    getAuthUrl: () => string;
    setAuthUrl: (url: string, force?: boolean) => void;
    getProxyPort: () => number;
    setProxyPort: (port: number) => void;
    getConfigPath: () => string;
    getAll: () => BobaConfig;
    getSessionToken: () => Promise<string | undefined>;
    setSessionToken: (token: string) => Promise<void>;
    clearSessionToken: () => Promise<void>;
    reset: () => void;
};
export type { AgentCredentials, AuthTokens, BobaConfig };
