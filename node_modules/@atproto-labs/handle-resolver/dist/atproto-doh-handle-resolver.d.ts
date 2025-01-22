import { AtprotoHandleResolver, AtprotoHandleResolverOptions } from './atproto-handle-resolver.js';
import { HandleResolver } from './types.js';
export type AtprotoDohHandleResolverOptions = Omit<AtprotoHandleResolverOptions, 'resolveTxt' | 'resolveTxtFallback'> & {
    dohEndpoint: string | URL;
};
export declare class AtprotoDohHandleResolver extends AtprotoHandleResolver implements HandleResolver {
    constructor(options: AtprotoDohHandleResolverOptions);
}
//# sourceMappingURL=atproto-doh-handle-resolver.d.ts.map