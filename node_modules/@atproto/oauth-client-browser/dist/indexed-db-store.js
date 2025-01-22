"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.IndexedDBStore = void 0;
const index_js_1 = require("./indexed-db/index.js");
const storeName = 'store';
class IndexedDBStore {
    constructor(dbName, maxAge = 600e3) {
        Object.defineProperty(this, "dbName", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: dbName
        });
        Object.defineProperty(this, "maxAge", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: maxAge
        });
    }
    async run(mode, fn) {
        const db = await index_js_1.DB.open(this.dbName, [
            (db) => {
                const store = db.createObjectStore(storeName);
                store.createIndex('createdAt', 'createdAt', { unique: false });
            },
        ], { durability: 'strict' });
        try {
            return await db.transaction([storeName], mode, (tx) => fn(tx.objectStore(storeName)));
        }
        finally {
            await db[Symbol.dispose]();
        }
    }
    async get(key) {
        const item = await this.run('readonly', (store) => store.get(key));
        if (!item)
            return undefined;
        const age = Date.now() - item.createdAt.getTime();
        if (age > this.maxAge) {
            await this.del(key);
            return undefined;
        }
        return item?.value;
    }
    async set(key, value) {
        await this.run('readwrite', (store) => {
            store.put({ value, createdAt: new Date() }, key);
        });
    }
    async del(key) {
        await this.run('readwrite', (store) => {
            store.delete(key);
        });
    }
    async deleteOutdated() {
        const upperBound = new Date(Date.now() - this.maxAge);
        const query = IDBKeyRange.upperBound(upperBound);
        await this.run('readwrite', async (store) => {
            const index = store.index('createdAt');
            const keys = await index.getAllKeys(query);
            for (const key of keys)
                store.delete(key);
        });
    }
}
exports.IndexedDBStore = IndexedDBStore;
//# sourceMappingURL=indexed-db-store.js.map