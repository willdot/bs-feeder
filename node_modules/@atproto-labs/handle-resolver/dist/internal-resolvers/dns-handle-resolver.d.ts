import { HandleResolver, ResolvedHandle } from '../types';
/**
 * DNS TXT record resolver. Return `null` if the hostname successfully does not
 * resolve to a valid DID. Throw an error if an unexpected error occurs.
 */
export type ResolveTxt = (hostname: string) => Promise<null | string[]>;
export declare class DnsHandleResolver implements HandleResolver {
    protected resolveTxt: ResolveTxt;
    constructor(resolveTxt: ResolveTxt);
    resolve(handle: string): Promise<ResolvedHandle>;
}
//# sourceMappingURL=dns-handle-resolver.d.ts.map