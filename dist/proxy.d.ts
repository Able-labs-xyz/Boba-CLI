interface ProxyServer {
    start: () => Promise<void>;
    stop: () => Promise<void>;
}
export declare function createProxyServer(): ProxyServer;
export {};
