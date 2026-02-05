interface ProxyServer {
    start: () => Promise<void>;
    stop: () => void;
}
export declare function createProxyServer(): ProxyServer;
export {};
