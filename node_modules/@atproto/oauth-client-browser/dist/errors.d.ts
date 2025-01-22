/**
 * Special error class destined to be thrown when the login process was
 * performed in a popup and should be continued in the parent/initiating window.
 */
export declare class LoginContinuedInParentWindowError extends Error {
    code: string;
    constructor();
}
//# sourceMappingURL=errors.d.ts.map