import { ResolvedHandle } from '@atproto-labs/handle-resolver';
import { SimpleStore, Value } from '@atproto-labs/simple-store';
import { DidDocument } from '@atproto/did';
import { InternalStateData, Session, TokenSet } from '@atproto/oauth-client';
import { OAuthAuthorizationServerMetadata, OAuthProtectedResourceMetadata } from '@atproto/oauth-types';
import { DBObjectStore } from './indexed-db/index.js';
type Item<V> = {
    value: V;
    expiresAt?: string;
};
type EncodedKey = {
    keyId: string;
    keyPair: CryptoKeyPair;
};
export type Schema = {
    state: Item<{
        dpopKey: EncodedKey;
        iss: string;
        verifier?: string;
        appState?: string;
    }>;
    session: Item<{
        dpopKey: EncodedKey;
        tokenSet: TokenSet;
    }>;
    didCache: Item<DidDocument>;
    dpopNonceCache: Item<string>;
    handleCache: Item<ResolvedHandle>;
    authorizationServerMetadataCache: Item<OAuthAuthorizationServerMetadata>;
    protectedResourceMetadataCache: Item<OAuthProtectedResourceMetadata>;
};
export type DatabaseStore<V extends Value> = SimpleStore<string, V>;
export type BrowserOAuthDatabaseOptions = {
    name?: string;
    durability?: 'strict' | 'relaxed';
    cleanupInterval?: number;
};
export declare class BrowserOAuthDatabase {
    #private;
    constructor(options?: BrowserOAuthDatabaseOptions);
    protected run<N extends keyof Schema, R>(storeName: N, mode: 'readonly' | 'readwrite', fn: (s: DBObjectStore<Schema[N]>) => R | Promise<R>): Promise<R>;
    protected createStore<N extends keyof Schema, V extends Value>(name: N, { encode, decode, expiresAt, }: {
        encode: (value: V) => Schema[N]['value'] | PromiseLike<Schema[N]['value']>;
        decode: (encoded: Schema[N]['value']) => V | PromiseLike<V>;
        expiresAt: (value: V) => null | Date;
    }): DatabaseStore<V>;
    getSessionStore(): DatabaseStore<Session>;
    getStateStore(): DatabaseStore<InternalStateData>;
    getDpopNonceCache(): undefined | DatabaseStore<string>;
    getDidCache(): undefined | DatabaseStore<DidDocument>;
    getHandleCache(): undefined | DatabaseStore<ResolvedHandle>;
    getAuthorizationServerMetadataCache(): undefined | DatabaseStore<OAuthAuthorizationServerMetadata>;
    getProtectedResourceMetadataCache(): undefined | DatabaseStore<OAuthProtectedResourceMetadata>;
    cleanup(): Promise<void>;
    [Symbol.asyncDispose](): Promise<void>;
}
export {};
//# sourceMappingURL=browser-oauth-database.d.ts.map