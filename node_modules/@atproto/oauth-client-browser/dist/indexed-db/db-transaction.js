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
var _DBTransaction_tx;
Object.defineProperty(exports, "__esModule", { value: true });
exports.DBTransaction = void 0;
const db_object_store_js_1 = require("./db-object-store.js");
class DBTransaction {
    constructor(tx) {
        _DBTransaction_tx.set(this, void 0);
        __classPrivateFieldSet(this, _DBTransaction_tx, tx, "f");
        const onAbort = () => {
            cleanup();
        };
        const onComplete = () => {
            cleanup();
        };
        const cleanup = () => {
            __classPrivateFieldSet(this, _DBTransaction_tx, null, "f");
            tx.removeEventListener('abort', onAbort);
            tx.removeEventListener('complete', onComplete);
        };
        tx.addEventListener('abort', onAbort);
        tx.addEventListener('complete', onComplete);
    }
    get tx() {
        if (!__classPrivateFieldGet(this, _DBTransaction_tx, "f"))
            throw new Error('Transaction already ended');
        return __classPrivateFieldGet(this, _DBTransaction_tx, "f");
    }
    async abort() {
        const { tx } = this;
        __classPrivateFieldSet(this, _DBTransaction_tx, null, "f");
        tx.abort();
    }
    async commit() {
        const { tx } = this;
        __classPrivateFieldSet(this, _DBTransaction_tx, null, "f");
        tx.commit?.();
    }
    objectStore(name) {
        const store = this.tx.objectStore(name);
        return new db_object_store_js_1.DBObjectStore(store);
    }
    [(_DBTransaction_tx = new WeakMap(), Symbol.dispose)]() {
        if (__classPrivateFieldGet(this, _DBTransaction_tx, "f"))
            this.commit();
    }
}
exports.DBTransaction = DBTransaction;
//# sourceMappingURL=db-transaction.js.map