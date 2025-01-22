import { AuthorizeOptions, Fetch, OAuthClient, OAuthClientOptions, OAuthSession } from '@atproto/oauth-client';
import { OAuthClientMetadataInput, OAuthResponseMode } from '@atproto/oauth-types';
import { Simplify } from './util.js';
export type BrowserOAuthClientOptions = Simplify<{
    clientMetadata?: Readonly<OAuthClientMetadataInput>;
    responseMode?: Exclude<OAuthResponseMode, 'form_post'>;
    fetch?: Fetch;
} & Omit<OAuthClientOptions, 'clientMetadata' | 'responseMode' | 'keyset' | 'fetch' | 'runtimeImplementation' | 'sessionStore' | 'stateStore' | 'didCache' | 'handleCache' | 'dpopNonceCache' | 'authorizationServerMetadataCache' | 'protectedResourceMetadataCache'>>;
export type BrowserOAuthClientLoadOptions = Simplify<{
    clientId: string;
    signal?: AbortSignal;
} & Omit<BrowserOAuthClientOptions, 'clientMetadata'>>;
export declare class BrowserOAuthClient extends OAuthClient implements Disposable {
    static load({ clientId, ...options }: BrowserOAuthClientLoadOptions): Promise<BrowserOAuthClient>;
    readonly [Symbol.dispose]: () => void;
    constructor({ clientMetadata, responseMode, ...options }: BrowserOAuthClientOptions);
    init(refresh?: boolean): Promise<{
        session: OAuthSession;
        state: string | null;
    } | {
        session: OAuthSession;
    } | undefined>;
    restore(sub: string, refresh?: boolean): Promise<OAuthSession>;
    revoke(sub: string): Promise<void>;
    signIn(input: string, options: AuthorizeOptions & {
        display: 'popup';
    }): Promise<OAuthSession>;
    signIn(input: string, options?: AuthorizeOptions): Promise<never>;
    signInRedirect(input: string, options?: AuthorizeOptions): Promise<never>;
    signInPopup(input: string, options?: Omit<AuthorizeOptions, 'state'>): Promise<OAuthSession>;
    private readCallbackParams;
    signInCallback(): Promise<{
        session: OAuthSession;
        state: string | null;
    } | null>;
    dispose(): void;
}
//# sourceMappingURL=browser-oauth-client.d.ts.map