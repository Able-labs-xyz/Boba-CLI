import axios from 'axios';
import { config } from './config.js';
import { logger } from './logger.js';
/**
 * Register agent with limit-orders service for trade execution.
 * This syncs the agent's wallets with TurnkeySigner.
 */
async function registerWithLimitOrders(tokens) {
    const authUrl = config.getAuthUrl();
    // Derive limit-orders URL from auth URL (same gateway)
    const limitUrl = authUrl.replace('/v2', '/v2/limit');
    try {
        logger.info('Registering with limit-orders service...');
        await axios.post(`${limitUrl}/agents/register`, {
            sub_organization_id: tokens.subOrganizationId,
            wallet_address: tokens.solanaAddress,
            evm_address: tokens.evmAddress,
        }, {
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${tokens.accessToken}`,
            },
            timeout: 30000,
        });
        logger.success('Registered with limit-orders service');
    }
    catch (error) {
        // Non-fatal: log warning but don't fail authentication
        if (axios.isAxiosError(error)) {
            const status = error.response?.status;
            const message = error.response?.data?.error || error.message;
            logger.warning(`Limit-orders registration failed (${status}): ${message}`);
        }
        else {
            logger.warning(`Limit-orders registration failed: ${error.message}`);
        }
    }
}
/**
 * Initialize wallet monitoring with portfolio service.
 * This registers the agent's wallets in Redis for real-time tracking.
 */
async function initializeWalletMonitoring(tokens) {
    const authUrl = config.getAuthUrl();
    // Derive portfolio URL from auth URL (same gateway)
    const portfolioUrl = authUrl.replace('/v2', '/v2/portfolio');
    try {
        logger.info('Initializing wallet monitoring...');
        const response = await axios.post(`${portfolioUrl}/${tokens.agentId}/wallets/init`, {}, {
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${tokens.accessToken}`,
            },
            timeout: 30000,
        });
        const data = response.data;
        if (data.subscribed > 0) {
            logger.success(`Wallet monitoring initialized (${data.subscribed} wallets)`);
        }
        else {
            logger.warning('No wallets found to monitor');
        }
    }
    catch (error) {
        // Non-fatal: log warning but don't fail authentication
        if (axios.isAxiosError(error)) {
            const status = error.response?.status;
            const message = error.response?.data?.error || error.message;
            logger.warning(`Wallet monitoring init failed (${status}): ${message}`);
        }
        else {
            logger.warning(`Wallet monitoring init failed: ${error.message}`);
        }
    }
}
export async function authenticate() {
    const credentials = await config.getCredentials();
    if (!credentials) {
        logger.error('No credentials found. Run "boba init" first.');
        return null;
    }
    const authUrl = config.getAuthUrl();
    try {
        logger.info('Authenticating with Boba backend...');
        const response = await axios.post(`${authUrl}/user/auth/authenticate`, {
            auth_method: 'agent',
            agent_id: credentials.agentId,
            agent_secret: credentials.agentSecret,
        }, {
            headers: { 'Content-Type': 'application/json' },
            timeout: 30000,
        });
        const authData = response.data.data;
        const tokens = {
            accessToken: authData.access_token,
            refreshToken: authData.refresh_token,
            accessTokenExpiresAt: authData.access_token_expires_at,
            refreshTokenExpiresAt: authData.refresh_token_expires_at,
            agentId: authData.agent_id,
            agentName: authData.agent_name,
            evmAddress: authData.evm_address,
            solanaAddress: authData.solana_address,
            subOrganizationId: authData.sub_organization_id,
        };
        config.setTokens(tokens);
        logger.success(`Authenticated as ${tokens.agentName} (${tokens.agentId.slice(0, 8)}...)`);
        // Register agent with limit-orders service for trade execution
        await registerWithLimitOrders(tokens);
        // Initialize wallet monitoring with portfolio service
        await initializeWalletMonitoring(tokens);
        return tokens;
    }
    catch (error) {
        if (axios.isAxiosError(error)) {
            const status = error.response?.status;
            const message = error.response?.data?.message || error.message;
            if (status === 401) {
                logger.error('Invalid credentials. Check your agent_id and secret.');
            }
            else if (status === 404) {
                logger.error('Agent not found. It may have been revoked.');
            }
            else {
                logger.error(`Authentication failed: ${message}`);
            }
        }
        else {
            logger.error(`Authentication failed: ${error.message}`);
        }
        return null;
    }
}
export async function refreshTokens() {
    const tokens = config.getTokens();
    if (!tokens?.refreshToken) {
        logger.warning('No refresh token available. Re-authenticating...');
        return authenticate();
    }
    const authUrl = config.getAuthUrl();
    try {
        logger.info('Refreshing access token...');
        const response = await axios.post(`${authUrl}/user/auth/refresh`, {
            refresh_token: tokens.refreshToken,
        }, {
            headers: { 'Content-Type': 'application/json' },
            timeout: 30000,
        });
        const refreshData = response.data.data;
        const newTokens = {
            ...tokens,
            accessToken: refreshData.access_token,
            accessTokenExpiresAt: refreshData.access_token_expires_at,
        };
        config.setTokens(newTokens);
        logger.success('Token refreshed');
        return newTokens;
    }
    catch (error) {
        logger.warning('Token refresh failed. Re-authenticating...');
        return authenticate();
    }
}
export async function ensureAuthenticated() {
    // Check if we have valid tokens
    const tokens = config.getTokens();
    if (!tokens) {
        return authenticate();
    }
    // Check if token is expired
    if (config.isTokenExpired()) {
        return refreshTokens();
    }
    return tokens;
}
export function getAccessToken() {
    const tokens = config.getTokens();
    return tokens?.accessToken || null;
}
