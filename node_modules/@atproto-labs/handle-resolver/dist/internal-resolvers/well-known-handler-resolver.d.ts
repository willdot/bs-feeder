import { HandleResolver, ResolveHandleOptions, ResolvedHandle } from '../types.js';
export type WellKnownHandleResolverOptions = {
    /**
     * Fetch function to use for HTTP requests. Allows customizing the request
     * behavior, e.g. adding headers, setting a timeout, mocking, etc. The
     * provided fetch function will be wrapped with a safeFetchWrap function that
     * adds SSRF protection.
     *
     * @default `globalThis.fetch`
     */
    fetch?: typeof globalThis.fetch;
};
export declare class WellKnownHandleResolver implements HandleResolver {
    protected readonly fetch: typeof globalThis.fetch;
    constructor(options?: WellKnownHandleResolverOptions);
    resolve(handle: string, options?: ResolveHandleOptions): Promise<ResolvedHandle>;
}
//# sourceMappingURL=well-known-handler-resolver.d.ts.map