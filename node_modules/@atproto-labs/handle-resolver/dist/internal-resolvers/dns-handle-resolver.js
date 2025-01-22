"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DnsHandleResolver = void 0;
const types_1 = require("../types");
const SUBDOMAIN = '_atproto';
const PREFIX = 'did=';
class DnsHandleResolver {
    constructor(resolveTxt) {
        Object.defineProperty(this, "resolveTxt", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: resolveTxt
        });
    }
    async resolve(handle) {
        const results = await this.resolveTxt.call(null, `${SUBDOMAIN}.${handle}`);
        if (!results)
            return null;
        for (let i = 0; i < results.length; i++) {
            // If the line does not start with "did=", skip it
            if (!results[i].startsWith(PREFIX))
                continue;
            // Ensure no other entry starting with "did=" follows
            for (let j = i + 1; j < results.length; j++) {
                if (results[j].startsWith(PREFIX))
                    return null;
            }
            // Note: No trimming (to be consistent with spec)
            const did = results[i].slice(PREFIX.length);
            // Invalid DBS record
            return (0, types_1.isResolvedHandle)(did) ? did : null;
        }
        return null;
    }
}
exports.DnsHandleResolver = DnsHandleResolver;
//# sourceMappingURL=dns-handle-resolver.js.map