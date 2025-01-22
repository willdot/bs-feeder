"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.OAuthCallbackError = void 0;
class OAuthCallbackError extends Error {
    static from(err, params, state) {
        if (err instanceof OAuthCallbackError)
            return err;
        const message = err instanceof Error ? err.message : undefined;
        return new OAuthCallbackError(params, message, state, err);
    }
    constructor(params, message = params.get('error_description') || 'OAuth callback error', state, cause) {
        super(message, { cause });
        Object.defineProperty(this, "params", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: params
        });
        Object.defineProperty(this, "state", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: state
        });
    }
}
exports.OAuthCallbackError = OAuthCallbackError;
//# sourceMappingURL=oauth-callback-error.js.map