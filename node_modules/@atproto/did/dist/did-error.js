"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.InvalidDidError = exports.DidError = void 0;
class DidError extends Error {
    constructor(did, message, code, status = 400, cause) {
        super(message, { cause });
        Object.defineProperty(this, "did", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: did
        });
        Object.defineProperty(this, "code", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: code
        });
        Object.defineProperty(this, "status", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: status
        });
    }
    /**
     * For compatibility with error handlers in common HTTP frameworks.
     */
    get statusCode() {
        return this.status;
    }
    toString() {
        return `${this.constructor.name} ${this.code} (${this.did}): ${this.message}`;
    }
    static from(cause, did) {
        if (cause instanceof DidError) {
            return cause;
        }
        const message = cause instanceof Error
            ? cause.message
            : typeof cause === 'string'
                ? cause
                : 'An unknown error occurred';
        const status = (typeof cause?.['statusCode'] === 'number'
            ? cause['statusCode']
            : undefined) ??
            (typeof cause?.['status'] === 'number' ? cause['status'] : undefined);
        return new DidError(did, message, 'did-unknown-error', status, cause);
    }
}
exports.DidError = DidError;
class InvalidDidError extends DidError {
    constructor(did, message, cause) {
        super(did, message, 'did-invalid', 400, cause);
    }
}
exports.InvalidDidError = InvalidDidError;
//# sourceMappingURL=did-error.js.map