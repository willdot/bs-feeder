import { Did, DidDocument } from '@atproto/did';
import { SimpleStoreMemory, SimpleStoreMemoryOptions } from '@atproto-labs/simple-store-memory';
import { DidCache } from './did-cache.js';
export type DidCacheMemoryOptions = SimpleStoreMemoryOptions<Did, DidDocument>;
export declare class DidCacheMemory extends SimpleStoreMemory<Did, DidDocument> implements DidCache {
    constructor(options?: DidCacheMemoryOptions);
}
//# sourceMappingURL=did-cache-memory.d.ts.map