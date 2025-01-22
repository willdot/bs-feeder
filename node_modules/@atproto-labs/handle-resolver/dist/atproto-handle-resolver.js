"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AtprotoHandleResolver = void 0;
const dns_handle_resolver_js_1 = require("./internal-resolvers/dns-handle-resolver.js");
const well_known_handler_resolver_js_1 = require("./internal-resolvers/well-known-handler-resolver.js");
const noop = () => { };
/**
 * Implementation of the official ATPROTO handle resolution strategy.
 * This implementation relies on two primitives:
 * - HTTP Well-Known URI resolution (requires a `fetch()` implementation)
 * - DNS TXT record resolution (requires a `resolveTxt()` function)
 */
class AtprotoHandleResolver {
    constructor(options) {
        Object.defineProperty(this, "httpResolver", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "dnsResolver", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        Object.defineProperty(this, "dnsResolverFallback", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        this.httpResolver = new well_known_handler_resolver_js_1.WellKnownHandleResolver(options);
        this.dnsResolver = new dns_handle_resolver_js_1.DnsHandleResolver(options.resolveTxt);
        this.dnsResolverFallback = options.resolveTxtFallback
            ? new dns_handle_resolver_js_1.DnsHandleResolver(options.resolveTxtFallback)
            : undefined;
    }
    async resolve(handle, options) {
        options?.signal?.throwIfAborted();
        const abortController = new AbortController();
        const { signal } = abortController;
        options?.signal?.addEventListener('abort', () => abortController.abort(), {
            signal,
        });
        const wrappedOptions = { ...options, signal };
        try {
            const dnsPromise = this.dnsResolver.resolve(handle, wrappedOptions);
            const httpPromise = this.httpResolver.resolve(handle, wrappedOptions);
            // Prevent uncaught promise rejection
            httpPromise.catch(noop);
            const dnsRes = await dnsPromise;
            if (dnsRes)
                return dnsRes;
            signal.throwIfAborted();
            const res = await httpPromise;
            if (res)
                return res;
            signal.throwIfAborted();
            return this.dnsResolverFallback?.resolve(handle, wrappedOptions) ?? null;
        }
        finally {
            // Cancel pending requests, and remove "abort" listener on incoming signal
            abortController.abort();
        }
    }
}
exports.AtprotoHandleResolver = AtprotoHandleResolver;
//# sourceMappingURL=atproto-handle-resolver.js.map