export type JWSAlgorithm = 'HS256' | 'HS384' | 'HS512' | 'PS256' | 'PS384' | 'PS512' | 'RS256' | 'RS384' | 'RS512' | 'ES256' | 'ES256K' | 'ES384' | 'ES512' | 'EdDSA';
export type SubtleAlgorithm = RsaHashedKeyGenParams | EcKeyGenParams;
export declare function toSubtleAlgorithm(alg: string, crv?: string, options?: {
    modulusLength?: number;
}): SubtleAlgorithm;
export declare function fromSubtleAlgorithm(algorithm: KeyAlgorithm): JWSAlgorithm;
export declare function isCryptoKeyPair(v: unknown, extractable?: boolean): v is CryptoKeyPair;
//# sourceMappingURL=util.d.ts.map