"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toSubtleAlgorithm = toSubtleAlgorithm;
exports.fromSubtleAlgorithm = fromSubtleAlgorithm;
exports.isCryptoKeyPair = isCryptoKeyPair;
function toSubtleAlgorithm(alg, crv, options) {
    switch (alg) {
        case 'PS256':
        case 'PS384':
        case 'PS512':
            return {
                name: 'RSA-PSS',
                hash: `SHA-${alg.slice(-3)}`,
                modulusLength: options?.modulusLength ?? 2048,
                publicExponent: new Uint8Array([0x01, 0x00, 0x01]),
            };
        case 'RS256':
        case 'RS384':
        case 'RS512':
            return {
                name: 'RSASSA-PKCS1-v1_5',
                hash: `SHA-${alg.slice(-3)}`,
                modulusLength: options?.modulusLength ?? 2048,
                publicExponent: new Uint8Array([0x01, 0x00, 0x01]),
            };
        case 'ES256':
        case 'ES384':
            return {
                name: 'ECDSA',
                namedCurve: `P-${alg.slice(-3)}`,
            };
        case 'ES512':
            return {
                name: 'ECDSA',
                namedCurve: 'P-521',
            };
        default:
            // https://github.com/w3c/webcrypto/issues/82#issuecomment-849856773
            throw new TypeError(`Unsupported alg "${alg}"`);
    }
}
function fromSubtleAlgorithm(algorithm) {
    switch (algorithm.name) {
        case 'RSA-PSS':
        case 'RSASSA-PKCS1-v1_5': {
            const hash = algorithm.hash.name;
            switch (hash) {
                case 'SHA-256':
                case 'SHA-384':
                case 'SHA-512': {
                    const prefix = algorithm.name === 'RSA-PSS' ? 'PS' : 'RS';
                    return `${prefix}${hash.slice(-3)}`;
                }
                default:
                    throw new TypeError('unsupported RsaHashedKeyAlgorithm hash');
            }
        }
        case 'ECDSA': {
            const namedCurve = algorithm.namedCurve;
            switch (namedCurve) {
                case 'P-256':
                case 'P-384':
                case 'P-512':
                    return `ES${namedCurve.slice(-3)}`;
                case 'P-521':
                    return 'ES512';
                default:
                    throw new TypeError('unsupported EcKeyAlgorithm namedCurve');
            }
        }
        case 'Ed448':
        case 'Ed25519':
            return 'EdDSA';
        default:
            // https://github.com/w3c/webcrypto/issues/82#issuecomment-849856773
            throw new TypeError(`Unexpected algorithm "${algorithm.name}"`);
    }
}
function isCryptoKeyPair(v, extractable) {
    return (typeof v === 'object' &&
        v !== null &&
        'privateKey' in v &&
        v.privateKey instanceof CryptoKey &&
        v.privateKey.type === 'private' &&
        (extractable == null || v.privateKey.extractable === extractable) &&
        v.privateKey.usages.includes('sign') &&
        'publicKey' in v &&
        v.publicKey instanceof CryptoKey &&
        v.publicKey.type === 'public' &&
        v.publicKey.extractable === true &&
        v.publicKey.usages.includes('verify'));
}
//# sourceMappingURL=util.js.map