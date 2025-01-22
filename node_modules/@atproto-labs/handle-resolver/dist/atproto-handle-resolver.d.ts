import { ResolveTxt } from './internal-resolvers/dns-handle-resolver.js';
import { WellKnownHandleResolverOptions } from './internal-resolvers/well-known-handler-resolver.js';
import { HandleResolver, ResolveHandleOptions, ResolvedHandle } from './types.js';
export type { ResolveTxt };
export type AtprotoHandleResolverOptions = WellKnownHandleResolverOptions & {
    resolveTxt: ResolveTxt;
    resolveTxtFallback?: ResolveTxt;
};
/**
 * Implementation of the official ATPROTO handle resolution strategy.
 * This implementation relies on two primitives:
 * - HTTP Well-Known URI resolution (requires a `fetch()` implementation)
 * - DNS TXT record resolution (requires a `resolveTxt()` function)
 */
export declare class AtprotoHandleResolver implements HandleResolver {
    private readonly httpResolver;
    private readonly dnsResolver;
    private readonly dnsResolverFallback?;
    constructor(options: AtprotoHandleResolverOptions);
    resolve(handle: string, options?: ResolveHandleOptions): Promise<ResolvedHandle>;
}
//# sourceMappingURL=atproto-handle-resolver.d.ts.map