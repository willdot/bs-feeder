import { DigestAlgorithm, Key, RuntimeImplementation, RuntimeLock } from '@atproto/oauth-client';
export declare class BrowserRuntimeImplementation implements RuntimeImplementation {
    requestLock: RuntimeLock | undefined;
    constructor();
    createKey(algs: string[]): Promise<Key>;
    getRandomValues(byteLength: number): Uint8Array;
    digest(data: Uint8Array, { name }: DigestAlgorithm): Promise<Uint8Array>;
}
//# sourceMappingURL=browser-runtime-implementation.d.ts.map