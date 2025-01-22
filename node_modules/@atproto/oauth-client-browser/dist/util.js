"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildLoopbackClientId = buildLoopbackClientId;
const oauth_types_1 = require("@atproto/oauth-types");
/**
 * @example
 * ```ts
 * const clientId = buildLoopbackClientId(window.location)
 * ```
 */
function buildLoopbackClientId(location, localhost = '127.0.0.1') {
    if (!(0, oauth_types_1.isLoopbackHost)(location.hostname)) {
        throw new TypeError(`Expected a loopback host, got ${location.hostname}`);
    }
    const redirectUri = `http://${location.hostname === 'localhost' ? localhost : location.hostname}${location.port && !location.port.startsWith(':') ? `:${location.port}` : location.port}${location.pathname}`;
    return `http://localhost${location.pathname === '/' ? '' : location.pathname}?redirect_uri=${encodeURIComponent(redirectUri)}`;
}
//# sourceMappingURL=util.js.map