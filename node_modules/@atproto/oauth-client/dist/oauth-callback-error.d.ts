export declare class OAuthCallbackError extends Error {
    readonly params: URLSearchParams;
    readonly state?: string | undefined;
    static from(err: unknown, params: URLSearchParams, state?: string): OAuthCallbackError;
    constructor(params: URLSearchParams, message?: string, state?: string | undefined, cause?: unknown);
}
//# sourceMappingURL=oauth-callback-error.d.ts.map