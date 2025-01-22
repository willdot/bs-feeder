import { Key } from '@atproto/jwk';
import { RuntimeImplementation, RuntimeLock } from './runtime-implementation.js';
export declare class Runtime {
    protected implementation: RuntimeImplementation;
    readonly hasImplementationLock: boolean;
    readonly usingLock: RuntimeLock;
    constructor(implementation: RuntimeImplementation);
    generateKey(algs: string[]): Promise<Key>;
    sha256(text: string): Promise<string>;
    generateNonce(length?: number): Promise<string>;
    generatePKCE(byteLength?: number): Promise<{
        verifier: string;
        challenge: string;
        method: "S256";
    }>;
    calculateJwkThumbprint(jwk: any): Promise<string>;
    /**
     * @see {@link https://datatracker.ietf.org/doc/html/rfc7636#section-4.1}
     * @note It is RECOMMENDED that the output of a suitable random number generator
     * be used to create a 32-octet sequence. The octet sequence is then
     * base64url-encoded to produce a 43-octet URL safe string to use as the code
     * verifier.
     */
    protected generateVerifier(byteLength?: number): Promise<string>;
}
//# sourceMappingURL=runtime.d.ts.map