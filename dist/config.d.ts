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
    getTokens: () => AuthTokens | undefined;
    setTokens: (tokens: AuthTokens) => void;
    clearTokens: () => void;
    isTokenExpired: () => boolean;
    getMcpUrl: () => string;
    setMcpUrl: (url: string) => void;
    getAuthUrl: () => string;
    setAuthUrl: (url: string) => void;
    getProxyPort: () => number;
    setProxyPort: (port: number) => void;
    getConfigPath: () => string;
    getAll: () => BobaConfig;
    reset: () => void;
};
export type { AgentCredentials, AuthTokens, BobaConfig };
