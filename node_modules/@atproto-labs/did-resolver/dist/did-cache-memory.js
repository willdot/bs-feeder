"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DidCacheMemory = void 0;
const simple_store_memory_1 = require("@atproto-labs/simple-store-memory");
const DEFAULT_TTL = 3600 * 1000; // 1 hour
const DEFAULT_MAX_SIZE = 50 * 1024 * 1024; // ~50MB
class DidCacheMemory extends simple_store_memory_1.SimpleStoreMemory {
    constructor(options) {
        super(options?.max == null
            ? { ttl: DEFAULT_TTL, maxSize: DEFAULT_MAX_SIZE, ...options }
            : { ttl: DEFAULT_TTL, ...options });
    }
}
exports.DidCacheMemory = DidCacheMemory;
//# sourceMappingURL=did-cache-memory.js.map