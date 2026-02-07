import Conf from 'conf';
import keytar from 'keytar';
import { logger } from './logger.js';

const KEYCHAIN_SERVICE = 'boba-cli';
const KEYCHAIN_ACCOUNT = 'agent-secret';
const KEYCHAIN_ACCESS_TOKEN = 'access-token';
const KEYCHAIN_REFRESH_TOKEN = 'refresh-token';
const KEYCHAIN_SESSION_TOKEN = 'session-token';

// In-memory cache for synchronous access in proxy hot path
let _cachedTokens: AuthTokens | undefined;

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

const defaultConfig: Partial<BobaConfig> = {
  mcpUrl: 'https://mcp-skunk.up.railway.app',
  authUrl: 'https://krakend-skunk.up.railway.app/v2',
  proxyPort: 3456,
  logLevel: 'info',
};

// URL allowlist to prevent credential exfiltration via social engineering
const ALLOWED_HOSTS = [
  'mcp-skunk.up.railway.app',
  'krakend-skunk.up.railway.app',
  'localhost',
  '127.0.0.1',
];

function isAllowedUrl(urlString: string): boolean {
  try {
    const url = new URL(urlString);
    return ALLOWED_HOSTS.includes(url.hostname);
  } catch {
    return false;
  }
}

// Config storage (non-sensitive data only - secrets go to OS keychain)
const store = new Conf<BobaConfig>({
  projectName: 'boba-cli',
  defaults: defaultConfig as BobaConfig,
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
  getCredentials: async (): Promise<AgentCredentials | undefined> => {
    const stored = store.get('credentials');
    if (!stored?.agentId) return undefined;

    // Try env var first (for CI/servers), then keychain
    let secret = process.env.BOBA_AGENT_SECRET;
    if (!secret) {
      try {
        secret = await keytar.getPassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCOUNT) || undefined;
      } catch {
        // Keychain not available, check legacy config
        secret = (stored as any).agentSecret;
      }
    }

    if (!secret) return undefined;

    return {
      agentId: stored.agentId,
      agentSecret: secret,
      name: stored.name,
    };
  },

  setCredentials: async (creds: AgentCredentials): Promise<void> => {
    // Store secret in OS keychain
    try {
      await keytar.setPassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCOUNT, creds.agentSecret);
    } catch (err) {
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

  clearCredentials: async (): Promise<void> => {
    store.delete('credentials');
    await config.clearTokens();
    try {
      await keytar.deletePassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCOUNT);
    } catch {
      // Ignore keychain errors
    }
    logger.info('Credentials cleared');
  },

  hasCredentials: (): boolean => {
    const creds = store.get('credentials');
    // Note: can't check keychain synchronously, just check if agentId exists
    return !!creds?.agentId;
  },

  // Tokens — sensitive values in OS keychain, metadata on disk
  getTokens: async (): Promise<AuthTokens | undefined> => {
    const meta = store.get('tokens') as any;
    if (!meta?.agentId) return undefined;

    // Backward-compatible: if legacy plaintext tokens exist, use them
    if (meta.accessToken) {
      _cachedTokens = meta as AuthTokens;
      return _cachedTokens;
    }

    try {
      const accessToken = await keytar.getPassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCESS_TOKEN);
      const refreshToken = await keytar.getPassword(KEYCHAIN_SERVICE, KEYCHAIN_REFRESH_TOKEN);
      if (!accessToken) return undefined;
      const tokens: AuthTokens = {
        accessToken,
        refreshToken: refreshToken || '',
        accessTokenExpiresAt: meta.accessTokenExpiresAt,
        refreshTokenExpiresAt: meta.refreshTokenExpiresAt,
        agentId: meta.agentId,
        agentName: meta.agentName,
        evmAddress: meta.evmAddress,
        solanaAddress: meta.solanaAddress,
        subOrganizationId: meta.subOrganizationId,
      };
      _cachedTokens = tokens;
      return tokens;
    } catch {
      // Keychain not available, fall back to legacy
      return meta.accessToken ? (meta as AuthTokens) : undefined;
    }
  },

  // Synchronous cache for proxy hot path (health endpoint, display)
  getTokensSync: (): AuthTokens | undefined => {
    return _cachedTokens || store.get('tokens') as AuthTokens | undefined;
  },

  setTokens: async (tokens: AuthTokens): Promise<void> => {
    _cachedTokens = tokens;
    try {
      await keytar.setPassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCESS_TOKEN, tokens.accessToken);
      await keytar.setPassword(KEYCHAIN_SERVICE, KEYCHAIN_REFRESH_TOKEN, tokens.refreshToken);
    } catch {
      logger.warning('Could not store tokens in OS keychain, using fallback');
      // Fallback: store everything in config (less secure, same as old behavior)
      store.set('tokens', tokens);
      return;
    }
    // Store only non-sensitive metadata on disk
    store.set('tokens', {
      accessTokenExpiresAt: tokens.accessTokenExpiresAt,
      refreshTokenExpiresAt: tokens.refreshTokenExpiresAt,
      agentId: tokens.agentId,
      agentName: tokens.agentName,
      evmAddress: tokens.evmAddress,
      solanaAddress: tokens.solanaAddress,
      subOrganizationId: tokens.subOrganizationId,
    } as any);
  },

  clearTokens: async (): Promise<void> => {
    _cachedTokens = undefined;
    store.delete('tokens');
    try {
      await keytar.deletePassword(KEYCHAIN_SERVICE, KEYCHAIN_ACCESS_TOKEN);
      await keytar.deletePassword(KEYCHAIN_SERVICE, KEYCHAIN_REFRESH_TOKEN);
    } catch { /* ignore */ }
  },

  isTokenExpired: (): boolean => {
    const tokens = store.get('tokens');
    if (!tokens?.accessTokenExpiresAt) return true;

    const expiresAt = new Date(tokens.accessTokenExpiresAt);
    const now = new Date();
    // Consider expired 1 minute before actual expiry
    return now.getTime() > expiresAt.getTime() - 60000;
  },

  // URLs
  getMcpUrl: (): string => {
    return store.get('mcpUrl') || defaultConfig.mcpUrl!;
  },

  setMcpUrl: (url: string, force = false): void => {
    if (!force && !isAllowedUrl(url)) {
      throw new Error(
        `Blocked: "${url}" is not an allowed MCP host. ` +
        `Allowed: ${ALLOWED_HOSTS.join(', ')}. Use --force to override.`
      );
    }
    store.set('mcpUrl', url);
  },

  getAuthUrl: (): string => {
    return store.get('authUrl') || defaultConfig.authUrl!;
  },

  setAuthUrl: (url: string, force = false): void => {
    if (!force && !isAllowedUrl(url)) {
      throw new Error(
        `Blocked: "${url}" is not an allowed auth host. ` +
        `Allowed: ${ALLOWED_HOSTS.join(', ')}. Use --force to override.`
      );
    }
    store.set('authUrl', url);
  },

  // Proxy
  getProxyPort: (): number => {
    return store.get('proxyPort') || defaultConfig.proxyPort!;
  },

  setProxyPort: (port: number): void => {
    store.set('proxyPort', port);
  },

  // Debug
  getConfigPath: (): string => {
    return store.path;
  },

  getAll: (): BobaConfig => {
    return store.store;
  },

  // Session token (per-session proxy auth, stored in OS keychain)
  getSessionToken: async (): Promise<string | undefined> => {
    try {
      const token = await keytar.getPassword(KEYCHAIN_SERVICE, KEYCHAIN_SESSION_TOKEN);
      return token || undefined;
    } catch {
      return undefined;
    }
  },

  setSessionToken: async (token: string): Promise<void> => {
    try {
      await keytar.setPassword(KEYCHAIN_SERVICE, KEYCHAIN_SESSION_TOKEN, token);
    } catch {
      logger.warning('Could not store session token in OS keychain');
    }
  },

  clearSessionToken: async (): Promise<void> => {
    try {
      await keytar.deletePassword(KEYCHAIN_SERVICE, KEYCHAIN_SESSION_TOKEN);
    } catch { /* ignore */ }
  },

  // Reset to defaults
  reset: (): void => {
    store.clear();
    logger.info('Config reset to defaults');
  },
};

export type { AgentCredentials, AuthTokens, BobaConfig };
