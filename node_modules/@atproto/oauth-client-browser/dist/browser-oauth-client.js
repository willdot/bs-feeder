"use strict";
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
exports.BrowserOAuthClient = void 0;
const oauth_client_1 = require("@atproto/oauth-client");
const oauth_types_1 = require("@atproto/oauth-types");
const browser_oauth_database_js_1 = require("./browser-oauth-database.js");
const browser_runtime_implementation_js_1 = require("./browser-runtime-implementation.js");
const errors_js_1 = require("./errors.js");
const util_js_1 = require("./util.js");
const NAMESPACE = `@@atproto/oauth-client-browser`;
//- Popup channel
const POPUP_CHANNEL_NAME = `${NAMESPACE}(popup-channel)`;
const POPUP_STATE_PREFIX = `${NAMESPACE}(popup-state):`;
const syncChannel = new BroadcastChannel(`${NAMESPACE}(synchronization-channel)`);
class BrowserOAuthClient extends oauth_client_1.OAuthClient {
    static async load({ clientId, ...options }) {
        if (clientId.startsWith('http:')) {
            const clientMetadata = (0, oauth_types_1.atprotoLoopbackClientMetadata)(clientId);
            return new BrowserOAuthClient({ clientMetadata, ...options });
        }
        else if (clientId.startsWith('https:')) {
            (0, oauth_types_1.assertOAuthDiscoverableClientId)(clientId);
            const clientMetadata = await oauth_client_1.OAuthClient.fetchMetadata({
                clientId,
                ...options,
            });
            return new BrowserOAuthClient({ clientMetadata, ...options });
        }
        else {
            throw new TypeError(`Invalid client id: ${clientId}`);
        }
    }
    constructor({ clientMetadata = (0, oauth_types_1.atprotoLoopbackClientMetadata)((0, util_js_1.buildLoopbackClientId)(window.location)), 
    // "fragment" is a safer default as the query params will not be sent to the server
    responseMode = 'fragment', ...options }) {
        if (!globalThis.crypto?.subtle) {
            throw new Error('WebCrypto API is required');
        }
        if (!['query', 'fragment'].includes(responseMode)) {
            // Make sure "form_post" is not used as it is not supported in the browser
            throw new TypeError(`Invalid response mode: ${responseMode}`);
        }
        const database = new browser_oauth_database_js_1.BrowserOAuthDatabase();
        super({
            ...options,
            clientMetadata,
            responseMode,
            keyset: undefined,
            runtimeImplementation: new browser_runtime_implementation_js_1.BrowserRuntimeImplementation(),
            sessionStore: database.getSessionStore(),
            stateStore: database.getStateStore(),
            didCache: database.getDidCache(),
            handleCache: database.getHandleCache(),
            dpopNonceCache: database.getDpopNonceCache(),
            authorizationServerMetadataCache: database.getAuthorizationServerMetadataCache(),
            protectedResourceMetadataCache: database.getProtectedResourceMetadataCache(),
        });
        Object.defineProperty(this, _a, {
            enumerable: true,
            configurable: true,
            writable: true,
            value: void 0
        });
        // TODO: replace with AsyncDisposableStack once they are standardized
        const ac = new AbortController();
        const { signal } = ac;
        this[Symbol.dispose] = () => ac.abort();
        signal.addEventListener('abort', () => database[Symbol.asyncDispose](), {
            once: true,
        });
        // Keep track of the current session
        this.addEventListener('deleted', ({ detail: { sub } }) => {
            if (localStorage.getItem(`${NAMESPACE}(sub)`) === sub) {
                localStorage.removeItem(`${NAMESPACE}(sub)`);
            }
        });
        // Session synchronization across tabs
        for (const type of ['deleted', 'updated']) {
            this.sessionGetter.addEventListener(type, ({ detail }) => {
                // Notify other tabs when a session is deleted or updated
                syncChannel.postMessage([type, detail]);
            });
        }
        syncChannel.addEventListener('message', (event) => {
            if (event.source !== window) {
                // Trigger listeners when an event is received from another tab
                const [type, detail] = event.data;
                this.dispatchCustomEvent(type, detail);
            }
        }, 
        // Remove the listener when the client is disposed
        { signal });
    }
    async init(refresh) {
        await fixLocation(this.clientMetadata);
        const signInResult = await this.signInCallback();
        if (signInResult) {
            localStorage.setItem(`${NAMESPACE}(sub)`, signInResult.session.sub);
            return signInResult;
        }
        const sub = localStorage.getItem(`${NAMESPACE}(sub)`);
        if (sub) {
            try {
                const session = await this.restore(sub, refresh);
                return { session };
            }
            catch (err) {
                localStorage.removeItem(`${NAMESPACE}(sub)`);
                throw err;
            }
        }
    }
    async restore(sub, refresh) {
        const session = await super.restore(sub, refresh);
        localStorage.setItem(`${NAMESPACE}(sub)`, session.sub);
        return session;
    }
    async revoke(sub) {
        localStorage.removeItem(`${NAMESPACE}(sub)`);
        return super.revoke(sub);
    }
    async signIn(input, options) {
        if (options?.display === 'popup') {
            return this.signInPopup(input, options);
        }
        else {
            return this.signInRedirect(input, options);
        }
    }
    async signInRedirect(input, options) {
        const url = await this.authorize(input, options);
        window.location.href = url.href;
        // back-forward cache
        return new Promise((resolve, reject) => {
            setTimeout((err) => {
                // Take the opportunity to proactively cancel the pending request
                this.abortRequest(url).then(() => reject(err), (reason) => reject(new AggregateError([err, reason])));
            }, 5e3, new Error('User navigated back'));
        });
    }
    async signInPopup(input, options) {
        // Open new window asap to prevent popup busting by browsers
        const popupFeatures = 'width=600,height=600,menubar=no,toolbar=no';
        let popup = window.open('about:blank', '_blank', popupFeatures);
        const stateKey = `${Math.random().toString(36).slice(2)}`;
        const url = await this.authorize(input, {
            ...options,
            state: `${POPUP_STATE_PREFIX}${stateKey}`,
            display: options?.display ?? 'popup',
        });
        options?.signal?.throwIfAborted();
        if (popup) {
            popup.window.location.href = url.href;
        }
        else {
            popup = window.open(url.href, '_blank', popupFeatures);
        }
        popup?.focus();
        return new Promise((resolve, reject) => {
            const popupChannel = new BroadcastChannel(POPUP_CHANNEL_NAME);
            const cleanup = () => {
                clearTimeout(timeout);
                popupChannel.removeEventListener('message', onMessage);
                popupChannel.close();
                options?.signal?.removeEventListener('abort', cancel);
                popup?.close();
            };
            const cancel = () => {
                // @TODO: Store fact that the request was cancelled, allowing any
                // callback (e.g. in the popup) to revoke the session or credentials.
                reject(new Error(options?.signal?.aborted ? 'Aborted' : 'Timeout'));
                cleanup();
            };
            options?.signal?.addEventListener('abort', cancel);
            const timeout = setTimeout(cancel, 5 * 60e3);
            const onMessage = async ({ data }) => {
                if (data.key !== stateKey)
                    return;
                if (!('result' in data))
                    return;
                // Send acknowledgment to popup window
                popupChannel.postMessage({ key: stateKey, ack: true });
                cleanup();
                const { result } = data;
                if (result.status === 'fulfilled') {
                    const sub = result.value;
                    try {
                        options?.signal?.throwIfAborted();
                        resolve(await this.restore(sub, false));
                    }
                    catch (err) {
                        reject(err);
                        void this.revoke(sub);
                    }
                }
                else {
                    const { message, params } = result.reason;
                    reject(new oauth_client_1.OAuthCallbackError(new URLSearchParams(params), message));
                }
            };
            popupChannel.addEventListener('message', onMessage);
        });
    }
    readCallbackParams() {
        const params = this.responseMode === 'fragment'
            ? new URLSearchParams(location.hash.slice(1))
            : new URLSearchParams(location.search);
        // Only if the current URL contains a valid oauth response params
        if (!params.has('state') || !(params.has('code') || params.has('error'))) {
            return null;
        }
        const matchesLocation = (url) => location.origin === url.origin && location.pathname === url.pathname;
        const redirectUrls = this.clientMetadata.redirect_uris.map((uri) => new URL(uri));
        // Only if the current URL is one of the redirect_uris
        if (!redirectUrls.some(matchesLocation))
            return null;
        return params;
    }
    async signInCallback() {
        const params = this.readCallbackParams();
        // Not a (valid) OAuth redirect
        if (!params)
            return null;
        // Replace the current history entry without the params (this will prevent
        // the following code to run again if the user refreshes the page)
        if (this.responseMode === 'fragment') {
            history.replaceState(null, '', location.pathname + location.search);
        }
        else if (this.responseMode === 'query') {
            history.replaceState(null, '', location.pathname);
        }
        // Utility function to send the result of the popup to the parent window
        const sendPopupResult = (message) => {
            const popupChannel = new BroadcastChannel(POPUP_CHANNEL_NAME);
            return new Promise((resolve) => {
                const cleanup = (result) => {
                    clearTimeout(timer);
                    popupChannel.removeEventListener('message', onMessage);
                    popupChannel.close();
                    resolve(result);
                };
                const onMessage = ({ data }) => {
                    if ('ack' in data && message.key === data.key)
                        cleanup(true);
                };
                popupChannel.addEventListener('message', onMessage);
                popupChannel.postMessage(message);
                // Receiving of "ack" should be very fast, giving it 500 ms anyway
                const timer = setTimeout(cleanup, 500, false);
            });
        };
        return this.callback(params)
            .then(async (result) => {
            if (result.state?.startsWith(POPUP_STATE_PREFIX)) {
                const receivedByParent = await sendPopupResult({
                    key: result.state.slice(POPUP_STATE_PREFIX.length),
                    result: {
                        status: 'fulfilled',
                        value: result.session.sub,
                    },
                });
                // Revoke the credentials if the parent window was closed
                if (!receivedByParent)
                    await result.session.signOut();
                throw new errors_js_1.LoginContinuedInParentWindowError(); // signInPopup
            }
            return result;
        })
            .catch(async (err) => {
            if (err instanceof oauth_client_1.OAuthCallbackError &&
                err.state?.startsWith(POPUP_STATE_PREFIX)) {
                await sendPopupResult({
                    key: err.state.slice(POPUP_STATE_PREFIX.length),
                    result: {
                        status: 'rejected',
                        reason: {
                            message: err.message,
                            params: Array.from(err.params.entries()),
                        },
                    },
                });
                throw new errors_js_1.LoginContinuedInParentWindowError(); // signInPopup
            }
            // Most probable cause at this point is that the "state" parameter is
            // invalid.
            throw err;
        })
            .catch((err) => {
            if (err instanceof errors_js_1.LoginContinuedInParentWindowError) {
                // parent will also try to close the popup
                window.close();
            }
            throw err;
        });
    }
    dispose() {
        this[Symbol.dispose]();
    }
}
exports.BrowserOAuthClient = BrowserOAuthClient;
_a = Symbol.dispose;
/**
 * Since "localhost" is often used either in IP mode or in hostname mode,
 * and because the redirect uris must use the IP mode, we need to make sure
 * that the current location url is not using "localhost".
 *
 * This is required for the IndexedDB to work properly. Indeed, the IndexedDB
 * is shared by origin, so we must ensure to be on the same origin as the
 * redirect uris.
 */
function fixLocation(clientMetadata) {
    if (!(0, oauth_types_1.isOAuthClientIdLoopback)(clientMetadata.client_id))
        return;
    if (window.location.hostname !== 'localhost')
        return;
    const locationUrl = new URL(window.location.href);
    for (const uri of clientMetadata.redirect_uris) {
        const url = new URL(uri);
        if ((url.hostname === '127.0.0.1' || url.hostname === '[::1]') &&
            (!url.port || url.port === locationUrl.port) &&
            url.protocol === locationUrl.protocol &&
            url.pathname === locationUrl.pathname) {
            url.port = locationUrl.port;
            window.location.href = url.href;
            // Prevent init() on the wrong origin
            throw new Error('Redirecting to loopback IP...');
        }
    }
    throw new Error(`Please use the loopback IP address instead of ${locationUrl}`);
}
//# sourceMappingURL=browser-oauth-client.js.map