"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.handleRequest = handleRequest;
exports.promisify = promisify;
function handleRequest(request, onSuccess, onError) {
    const cleanup = () => {
        request.removeEventListener('success', success);
        request.removeEventListener('error', error);
    };
    const success = () => {
        onSuccess(request.result);
        cleanup();
    };
    const error = () => {
        onError(request.error || new Error('Unknown error'));
        cleanup();
    };
    request.addEventListener('success', success);
    request.addEventListener('error', error);
}
function promisify(request) {
    return new Promise((resolve, reject) => {
        handleRequest(request, resolve, reject);
    });
}
//# sourceMappingURL=util.js.map