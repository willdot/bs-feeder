"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DBIndex = void 0;
const util_js_1 = require("./util.js");
class DBIndex {
    constructor(idbIndex) {
        Object.defineProperty(this, "idbIndex", {
            enumerable: true,
            configurable: true,
            writable: true,
            value: idbIndex
        });
    }
    count(query) {
        return (0, util_js_1.promisify)(this.idbIndex.count(query));
    }
    get(query) {
        return (0, util_js_1.promisify)(this.idbIndex.get(query));
    }
    getKey(query) {
        return (0, util_js_1.promisify)(this.idbIndex.getKey(query));
    }
    getAll(query, count) {
        return (0, util_js_1.promisify)(this.idbIndex.getAll(query, count));
    }
    getAllKeys(query, count) {
        return (0, util_js_1.promisify)(this.idbIndex.getAllKeys(query, count));
    }
    deleteAll(query) {
        return new Promise((resolve, reject) => {
            const result = this.idbIndex.openCursor(query);
            result.onsuccess = function (event) {
                const cursor = event.target.result;
                if (cursor) {
                    cursor.delete();
                    cursor.continue();
                }
                else {
                    resolve();
                }
            };
            result.onerror = function (event) {
                reject(event.target?.error || new Error('Unexpected error'));
            };
        });
    }
}
exports.DBIndex = DBIndex;
//# sourceMappingURL=db-index.js.map