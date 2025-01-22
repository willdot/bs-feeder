"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenInvalidError = void 0;
class TokenInvalidError extends Error {
    constructor(sub, message = `The session for "${sub}" is invalid`, options) {
        super(message, options);
        Object.defineProperty(this, "sub", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: sub
        });
    }
}
exports.TokenInvalidError = TokenInvalidError;
//# sourceMappingURL=token-invalid-error.js.map