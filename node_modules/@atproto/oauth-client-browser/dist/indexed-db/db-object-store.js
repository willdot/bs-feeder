"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DBObjectStore = void 0;
const db_index_js_1 = require("./db-index.js");
const util_js_1 = require("./util.js");
class DBObjectStore {
    constructor(idbObjStore) {
        Object.defineProperty(this, "idbObjStore", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: idbObjStore
        });
    }
    get name() {
        return this.idbObjStore.name;
    }
    index(name) {
        return new db_index_js_1.DBIndex(this.idbObjStore.index(name));
    }
    get(key) {
        return (0, util_js_1.promisify)(this.idbObjStore.get(key));
    }
    getKey(query) {
        return (0, util_js_1.promisify)(this.idbObjStore.getKey(query));
    }
    getAll(query, count) {
        return (0, util_js_1.promisify)(this.idbObjStore.getAll(query, count));
    }
    getAllKeys(query, count) {
        return (0, util_js_1.promisify)(this.idbObjStore.getAllKeys(query, count));
    }
    add(value, key) {
        return (0, util_js_1.promisify)(this.idbObjStore.add(value, key));
    }
    put(value, key) {
        return (0, util_js_1.promisify)(this.idbObjStore.put(value, key));
    }
    delete(key) {
        return (0, util_js_1.promisify)(this.idbObjStore.delete(key));
    }
    clear() {
        return (0, util_js_1.promisify)(this.idbObjStore.clear());
    }
}
exports.DBObjectStore = DBObjectStore;
//# sourceMappingURL=db-object-store.js.map