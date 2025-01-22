"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenRefreshError = void 0;
class TokenRefreshError extends Error {
    constructor(sub, message, options) {
        super(message, options);
        Object.defineProperty(this, "sub", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: sub
        });
    }
}
exports.TokenRefreshError = TokenRefreshError;
//# sourceMappingURL=token-refresh-error.js.map