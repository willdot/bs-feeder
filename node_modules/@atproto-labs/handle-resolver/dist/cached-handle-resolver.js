"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CachedHandleResolver = void 0;
const simple_store_1 = require("@atproto-labs/simple-store");
const simple_store_memory_1 = require("@atproto-labs/simple-store-memory");
class CachedHandleResolver {
    constructor(
    /**
     * The resolver that will be used to resolve handles.
     */
    resolver, cache = new simple_store_memory_1.SimpleStoreMemory({
        max: 1000,
        ttl: 10 * 60e3,
    })) {
        Object.defineProperty(this, "getter", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        this.getter = new simple_store_1.CachedGetter((handle, options) => resolver.resolve(handle, options), cache);
    }
    async resolve(handle, options) {
        return this.getter.get(handle, options);
    }
}
exports.CachedHandleResolver = CachedHandleResolver;
//# sourceMappingURL=cached-handle-resolver.js.map