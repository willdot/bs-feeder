"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.OAuthResolverError = void 0;
const zod_1 = require("zod");
class OAuthResolverError extends Error {
    constructor(message, options) {
        super(message, options);
    }
    static from(cause, message) {
        if (cause instanceof OAuthResolverError)
            return cause;
        const validationReason = cause instanceof zod_1.ZodError
            ? `${cause.errors[0].path} ${cause.errors[0].message}`
            : null;
        const fullMessage = (message ?? `Unable to resolve identity`) +
            (validationReason ? ` (${validationReason})` : '');
        return new OAuthResolverError(fullMessage, {
            cause,
        });
    }
}
exports.OAuthResolverError = OAuthResolverError;
//# sourceMappingURL=oauth-resolver-error.js.map