"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WellKnownHandleResolver = void 0;
const types_js_1 = require("../types.js");
class WellKnownHandleResolver {
    constructor(options) {
        Object.defineProperty(this, "fetch", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        this.fetch = options?.fetch ?? globalThis.fetch;
    }
    async resolve(handle, options) {
        const url = new URL('/.well-known/atproto-did', `https://${handle}`);
        try {
            const response = await this.fetch.call(null, url, {
                cache: options?.noCache ? 'no-cache' : undefined,
                signal: options?.signal,
                redirect: 'error',
            });
            const text = await response.text();
            const firstLine = text.split('\n')[0].trim();
            if ((0, types_js_1.isResolvedHandle)(firstLine))
                return firstLine;
            return null;
        }
        catch (err) {
            // The the request failed, assume the handle does not resolve to a DID,
            // unless the failure was due to the signal being aborted.
            options?.signal?.throwIfAborted();
            return null;
        }
    }
}
exports.WellKnownHandleResolver = WellKnownHandleResolver;
//# sourceMappingURL=well-known-handler-resolver.js.map