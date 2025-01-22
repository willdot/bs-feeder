"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DidResolverCached = void 0;
const simple_store_1 = require("@atproto-labs/simple-store");
const did_cache_memory_js_1 = require("./did-cache-memory.js");
class DidResolverCached {
    constructor(resolver, cache = new did_cache_memory_js_1.DidCacheMemory()) {
        Object.defineProperty(this, "getter", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        this.getter = new simple_store_1.CachedGetter((did, options) => resolver.resolve(did, options), cache);
    }
    async resolve(did, options) {
        return this.getter.get(did, options);
    }
}
exports.DidResolverCached = DidResolverCached;
//# sourceMappingURL=did-cache.js.map