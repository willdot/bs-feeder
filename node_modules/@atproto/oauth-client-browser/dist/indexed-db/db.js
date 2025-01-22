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
var _DB_db;
Object.defineProperty(exports, "__esModule", { value: true });
exports.DB = void 0;
const db_transaction_js_1 = require("./db-transaction.js");
class DB {
    static async open(dbName, migrations, txOptions) {
        const db = await new Promise((resolve, reject) => {
            const request = indexedDB.open(dbName, migrations.length);
            request.onerror = () => reject(request.error);
            request.onsuccess = () => resolve(request.result);
            request.onupgradeneeded = ({ oldVersion, newVersion }) => {
                const db = request.result;
                try {
                    for (let version = oldVersion; version < (newVersion ?? migrations.length); ++version) {
                        const migration = migrations[version];
                        if (migration)
                            migration(db);
                        else
                            throw new Error(`Missing migration for version ${version}`);
                    }
                }
                catch (err) {
                    db.close();
                    reject(err);
                }
            };
        });
        return new DB(db, txOptions);
    }
    constructor(db, txOptions) {
        Object.defineProperty(this, "txOptions", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: txOptions
        });
        _DB_db.set(this, void 0);
        __classPrivateFieldSet(this, _DB_db, db, "f");
        const cleanup = () => {
            __classPrivateFieldSet(this, _DB_db, null, "f");
            db.removeEventListener('versionchange', cleanup);
            db.removeEventListener('close', cleanup);
            db.close(); // Can we call close on a "closed" database?
        };
        db.addEventListener('versionchange', cleanup);
        db.addEventListener('close', cleanup);
    }
    get db() {
        if (!__classPrivateFieldGet(this, _DB_db, "f"))
            throw new Error('Database closed');
        return __classPrivateFieldGet(this, _DB_db, "f");
    }
    get name() {
        return this.db.name;
    }
    get objectStoreNames() {
        return this.db.objectStoreNames;
    }
    get version() {
        return this.db.version;
    }
    async transaction(storeNames, mode, run) {
        // eslint-disable-next-line no-async-promise-executor
        return new Promise(async (resolve, reject) => {
            try {
                const tx = this.db.transaction(storeNames, mode, this.txOptions);
                let result = { done: false };
                tx.oncomplete = () => {
                    if (result.done)
                        resolve(result.value);
                    else
                        reject(new Error('Transaction completed without result'));
                };
                tx.onerror = () => reject(tx.error);
                tx.onabort = () => reject(tx.error || new Error('Transaction aborted'));
                try {
                    const value = await run(new db_transaction_js_1.DBTransaction(tx));
                    result = { done: true, value };
                    tx.commit();
                }
                catch (err) {
                    tx.abort();
                    throw err;
                }
            }
            catch (err) {
                reject(err);
            }
        });
    }
    close() {
        const { db } = this;
        __classPrivateFieldSet(this, _DB_db, null, "f");
        db.close();
    }
    [(_DB_db = new WeakMap(), Symbol.dispose)]() {
        if (__classPrivateFieldGet(this, _DB_db, "f"))
            return this.close();
    }
}
exports.DB = DB;
//# sourceMappingURL=db.js.map