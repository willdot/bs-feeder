"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenRevokedError = void 0;
class TokenRevokedError extends Error {
    constructor(sub, message = `The session for "${sub}" was successfully revoked`, options) {
        super(message, options);
        Object.defineProperty(this, "sub", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: sub
        });
    }
}
exports.TokenRevokedError = TokenRevokedError;
//# sourceMappingURL=token-revoked-error.js.map