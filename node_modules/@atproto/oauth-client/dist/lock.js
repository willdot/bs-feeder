"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.requestLocalLock = void 0;
const locks = new Map();
function acquireLocalLock(name) {
    return new Promise((resolveAcquire) => {
        const prev = locks.get(name) ?? Promise.resolve();
        const next = prev.then(() => {
            return new Promise((resolveRelease) => {
                const release = () => {
                    // Only delete the lock if it is still the current one
                    if (locks.get(name) === next)
                        locks.delete(name);
                    resolveRelease();
                };
                resolveAcquire(release);
            });
        });
        locks.set(name, next);
    });
}
const requestLocalLock = (name, fn) => {
    return acquireLocalLock(name).then(async (release) => {
        try {
            return await fn();
        }
        finally {
            release();
        }
    });
};
exports.requestLocalLock = requestLocalLock;
//# sourceMappingURL=lock.js.map