"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.LoginContinuedInParentWindowError = void 0;
/**
 * Special error class destined to be thrown when the login process was
 * performed in a popup and should be continued in the parent/initiating window.
 */
class LoginContinuedInParentWindowError extends Error {
    constructor() {
        super('Login complete, please close the popup window.');
        Object.defineProperty(this, "code", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: 'LOGIN_CONTINUED_IN_PARENT_WINDOW'
        });
    }
}
exports.LoginContinuedInParentWindowError = LoginContinuedInParentWindowError;
//# sourceMappingURL=errors.js.map