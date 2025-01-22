"use strict";
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _BrowserOAuthDatabase_dbPromise, _BrowserOAuthDatabase_cleanupInterval;
Object.defineProperty(exports, "__esModule", { value: true });
exports.BrowserOAuthDatabase = void 0;
const jwk_webcrypto_1 = require("@atproto/jwk-webcrypto");
const index_js_1 = require("./indexed-db/index.js");
function encodeKey(key) {
    if (!(key instanceof jwk_webcrypto_1.WebcryptoKey) || !key.kid) {
        throw new Error('Invalid key object');
    }
    return {
        keyId: key.kid,
        keyPair: key.cryptoKeyPair,
    };
}
async function decodeKey(encoded) {
    return jwk_webcrypto_1.WebcryptoKey.fromKeypair(encoded.keyPair, encoded.keyId);
}
const STORES = [
    'state',
    'session',
    'didCache',
    'dpopNonceCache',
    'handleCache',
    'authorizationServerMetadataCache',
    'protectedResourceMetadataCache',
];
class BrowserOAuthDatabase {
    constructor(options) {
        _BrowserOAuthDatabase_dbPromise.set(this, void 0);
        _BrowserOAuthDatabase_cleanupInterval.set(this, void 0);
        __classPrivateFieldSet(this, _BrowserOAuthDatabase_dbPromise, index_js_1.DB.open(options?.name ?? '@atproto-oauth-client', [
            (db) => {
                for (const name of STORES) {
                    const store = db.createObjectStore(name, { autoIncrement: true });
                    store.createIndex('expiresAt', 'expiresAt', { unique: false });
                }
            },
        ], { durability: options?.durability ?? 'strict' }), "f");
        __classPrivateFieldSet(this, _BrowserOAuthDatabase_cleanupInterval, setInterval(() => {
            void this.cleanup();
        }, options?.cleanupInterval ?? 30e3), "f");
    }
    async run(storeName, mode, fn) {
        const db = await __classPrivateFieldGet(this, _BrowserOAuthDatabase_dbPromise, "f");
        return await db.transaction([storeName], mode, (tx) => fn(tx.objectStore(storeName)));
    }
    createStore(name, { encode, decode, expiresAt, }) {
        return {
            get: async (key) => {
                // Find item in store
                const item = await this.run(name, 'readonly', (store) => store.get(key));
                // Not found
                if (item === undefined)
                    return undefined;
                // Too old (delete)
                if (item.expiresAt != null && new Date(item.expiresAt) < new Date()) {
                    await this.run(name, 'readwrite', (store) => store.delete(key));
                    return undefined;
                }
                // Item found and valid. Decode
                return decode(item.value);
            },
            set: async (key, value) => {
                // Create encoded item record
                const item = {
                    value: await encode(value),
                    expiresAt: expiresAt(value)?.toISOString(),
                };
                // Store item record
                await this.run(name, 'readwrite', (store) => store.put(item, key));
            },
            del: async (key) => {
                // Delete
                await this.run(name, 'readwrite', (store) => store.delete(key));
            },
        };
    }
    getSessionStore() {
        return this.createStore('session', {
            expiresAt: ({ tokenSet }) => tokenSet.refresh_token || tokenSet.expires_at == null
                ? null
                : new Date(tokenSet.expires_at),
            encode: ({ dpopKey, ...session }) => ({
                ...session,
                dpopKey: encodeKey(dpopKey),
            }),
            decode: async ({ dpopKey, ...encoded }) => ({
                ...encoded,
                dpopKey: await decodeKey(dpopKey),
            }),
        });
    }
    getStateStore() {
        return this.createStore('state', {
            expiresAt: (_value) => new Date(Date.now() + 10 * 60e3),
            encode: ({ dpopKey, ...session }) => ({
                ...session,
                dpopKey: encodeKey(dpopKey),
            }),
            decode: async ({ dpopKey, ...encoded }) => ({
                ...encoded,
                dpopKey: await decodeKey(dpopKey),
            }),
        });
    }
    getDpopNonceCache() {
        return this.createStore('dpopNonceCache', {
            expiresAt: (_value) => new Date(Date.now() + 600e3),
            encode: (value) => value,
            decode: (encoded) => encoded,
        });
    }
    getDidCache() {
        return this.createStore('didCache', {
            expiresAt: (_value) => new Date(Date.now() + 60e3),
            encode: (value) => value,
            decode: (encoded) => encoded,
        });
    }
    getHandleCache() {
        return this.createStore('handleCache', {
            expiresAt: (_value) => new Date(Date.now() + 60e3),
            encode: (value) => value,
            decode: (encoded) => encoded,
        });
    }
    getAuthorizationServerMetadataCache() {
        return this.createStore('authorizationServerMetadataCache', {
            expiresAt: (_value) => new Date(Date.now() + 60e3),
            encode: (value) => value,
            decode: (encoded) => encoded,
        });
    }
    getProtectedResourceMetadataCache() {
        return this.createStore('protectedResourceMetadataCache', {
            expiresAt: (_value) => new Date(Date.now() + 60e3),
            encode: (value) => value,
            decode: (encoded) => encoded,
        });
    }
    async cleanup() {
        const db = await __classPrivateFieldGet(this, _BrowserOAuthDatabase_dbPromise, "f");
        for (const name of STORES) {
            await db.transaction([name], 'readwrite', (tx) => tx
                .objectStore(name)
                .index('expiresAt')
                .deleteAll(IDBKeyRange.upperBound(Date.now())));
        }
    }
    async [(_BrowserOAuthDatabase_dbPromise = new WeakMap(), _BrowserOAuthDatabase_cleanupInterval = new WeakMap(), Symbol.asyncDispose)]() {
        clearInterval(__classPrivateFieldGet(this, _BrowserOAuthDatabase_cleanupInterval, "f"));
        __classPrivateFieldSet(this, _BrowserOAuthDatabase_cleanupInterval, undefined, "f");
        const dbPromise = __classPrivateFieldGet(this, _BrowserOAuthDatabase_dbPromise, "f");
        __classPrivateFieldSet(this, _BrowserOAuthDatabase_dbPromise, Promise.reject(new Error('Database has been disposed')), "f");
        // Avoid "unhandled promise rejection"
        __classPrivateFieldGet(this, _BrowserOAuthDatabase_dbPromise, "f").catch(() => null);
        // Spec recommends not to throw errors in dispose
        const db = await dbPromise.catch(() => null);
        if (db)
            await (db[Symbol.asyncDispose] || db[Symbol.dispose]).call(db);
    }
}
exports.BrowserOAuthDatabase = BrowserOAuthDatabase;
//# sourceMappingURL=browser-oauth-database.js.map