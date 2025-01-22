export declare class DidError extends Error {
    readonly did: string;
    readonly code: string;
    readonly status: number;
    constructor(did: string, message: string, code: string, status?: number, cause?: unknown);
    /**
     * For compatibility with error handlers in common HTTP frameworks.
     */
    get statusCode(): number;
    toString(): string;
    static from(cause: unknown, did: string): DidError;
}
export declare class InvalidDidError extends DidError {
    constructor(did: string, message: string, cause?: unknown);
}
//# sourceMappingURL=did-error.d.ts.map