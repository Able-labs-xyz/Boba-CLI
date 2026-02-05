import Conf from 'conf';
import keytar from 'keytar';
import { logger } from './logger.js';
const KEYCHAIN_SERVICE = 'boba-cli';
const KEYCHAIN_ACCOUNT = 'agent-secret';
const defaultConfig = {
    mcpUrl: 'https://mcp-skunk.up.railway.app',
    authUrl: 'https://krakend-skunk.up.railway.app/v2',
    proxyPort: 3456,
    logLevel: 'info',
};
// Config storage (non-sensitive data only - secrets go to OS keychain)
const store = new Conf({
    projectName: 'boba-cli',
    defaults: defaultConfig,
    schema: {
        credentials: {
            type: 'object',
            properties: {
                agentId: { type: 'string' },
                // agentSecret stored in OS keychain, not here
                name: { type: 'string' },
            },
        },
        tokens: {
            type: 'object',
            properties: {
                accessToken: { type: 'string' },
                refreshToken: { type: 'string' },
                accessTokenExpiresAt: { type: 'string' },
                refreshTokenExpiresAt: { type: 'string' },
                agentId: { type: 'string' },
                agentName: { type: 'string' },
                evmAddress: { type: 'string' },
                solanaAddress: { type: 'string' },
                subOrganizationId: { type: 'string' },
            },
        },
        mcpUrl: { type: 'string' },
        authUrl: { type: 'string' },
        proxyPort: { type: 'number' },
        logLevel: { type: 'string', enum: ['debug', 'info', 'warn', 'error'] },
    },
});
export const config = {
    // Credentials - agentSecret stored in OS keychain for security
    getCredentials: async () => {
        const stored = store.get('credentials');
        if (!stored?.agentId)
            return undefined;
        // Try env var first (for CI/servers), then keychain
        let secret = process.env.BOBA_AGENT_SECRET;
        if (!secret) {
            try {
                secret = await keytar.getPassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCOUNT) || undefined;
            }
            catch {
                // Keychain not available, check legacy config
                secret = stored.agentSecret;
            }
        }
        if (!secret)
            return undefined;
        return {
            agentId: stored.agentId,
            agentSecret: secret,
            name: stored.name,
        };
    },
    setCredentials: async (creds) => {
        // Store secret in OS keychain
        try {
            await keytar.setPassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCOUNT, creds.agentSecret);
        }
        catch (err) {
            logger.warning('Could not store secret in OS keychain, using fallback');
            // Fallback: store in config (less secure)
            store.set('credentials', creds);
            return;
        }
        // Store non-sensitive data in config file
        store.set('credentials', {
            agentId: creds.agentId,
            name: creds.name,
        });
        logger.success(`Credentials saved for agent: ${creds.name || creds.agentId.slice(0, 8)}...`);
    },
    clearCredentials: async () => {
        store.delete('credentials');
        store.delete('tokens');
        try {
            await keytar.deletePassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCOUNT);
        }
        catch {
            // Ignore keychain errors
        }
        logger.info('Credentials cleared');
    },
    hasCredentials: () => {
        const creds = store.get('credentials');
        // Note: can't check keychain synchronously, just check if agentId exists
        return !!creds?.agentId;
    },
    // Tokens
    getTokens: () => {
        return store.get('tokens');
    },
    setTokens: (tokens) => {
        store.set('tokens', tokens);
    },
    clearTokens: () => {
        store.delete('tokens');
    },
    isTokenExpired: () => {
        const tokens = store.get('tokens');
        if (!tokens?.accessTokenExpiresAt)
            return true;
        const expiresAt = new Date(tokens.accessTokenExpiresAt);
        const now = new Date();
        // Consider expired 1 minute before actual expiry
        return now.getTime() > expiresAt.getTime() - 60000;
    },
    // URLs
    getMcpUrl: () => {
        return store.get('mcpUrl') || defaultConfig.mcpUrl;
    },
    setMcpUrl: (url) => {
        store.set('mcpUrl', url);
    },
    getAuthUrl: () => {
        return store.get('authUrl') || defaultConfig.authUrl;
    },
    setAuthUrl: (url) => {
        store.set('authUrl', url);
    },
    // Proxy
    getProxyPort: () => {
        return store.get('proxyPort') || defaultConfig.proxyPort;
    },
    setProxyPort: (port) => {
        store.set('proxyPort', port);
    },
    // Debug
    getConfigPath: () => {
        return store.path;
    },
    getAll: () => {
        return store.store;
    },
    // Reset to defaults
    reset: () => {
        store.clear();
        logger.info('Config reset to defaults');
    },
};
