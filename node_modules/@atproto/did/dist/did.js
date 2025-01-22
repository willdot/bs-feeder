"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.didSchema = exports.DID_PREFIX = void 0;
exports.assertDidMethod = assertDidMethod;
exports.extractDidMethod = extractDidMethod;
exports.assertDidMsid = assertDidMsid;
exports.assertDid = assertDid;
exports.isDid = isDid;
exports.asDid = asDid;
const zod_1 = require("zod");
const did_error_js_1 = require("./did-error.js");
const DID_PREFIX = 'did:';
exports.DID_PREFIX = DID_PREFIX;
const DID_PREFIX_LENGTH = DID_PREFIX.length;
/**
 * DID Method-name check function.
 *
 * Check if the input is a valid DID method name, at the position between
 * `start` (inclusive) and `end` (exclusive).
 */
function assertDidMethod(input, start = 0, end = input.length) {
    if (!Number.isFinite(end) ||
        !Number.isFinite(start) ||
        end < start ||
        end > input.length) {
        throw new TypeError('Invalid start or end position');
    }
    if (end === start) {
        throw new did_error_js_1.InvalidDidError(input, `Empty method name`);
    }
    let c;
    for (let i = start; i < end; i++) {
        c = input.charCodeAt(i);
        if ((c < 0x61 || c > 0x7a) && // a-z
            (c < 0x30 || c > 0x39) // 0-9
        ) {
            throw new did_error_js_1.InvalidDidError(input, `Invalid character at position ${i} in DID method name`);
        }
    }
}
/**
 * This method assumes the input is a valid Did
 */
function extractDidMethod(did) {
    const msidSep = did.indexOf(':', DID_PREFIX_LENGTH);
    const method = did.slice(DID_PREFIX_LENGTH, msidSep);
    return method;
}
/**
 * DID Method-specific identifier check function.
 *
 * Check if the input is a valid DID method-specific identifier, at the position
 * between `start` (inclusive) and `end` (exclusive).
 */
function assertDidMsid(input, start = 0, end = input.length) {
    if (!Number.isFinite(end) ||
        !Number.isFinite(start) ||
        end < start ||
        end > input.length) {
        throw new TypeError('Invalid start or end position');
    }
    if (end === start) {
        throw new did_error_js_1.InvalidDidError(input, `DID method-specific id must not be empty`);
    }
    let c;
    for (let i = start; i < end; i++) {
        c = input.charCodeAt(i);
        // Check for frequent chars first
        if ((c < 0x61 || c > 0x7a) && // a-z
            (c < 0x41 || c > 0x5a) && // A-Z
            (c < 0x30 || c > 0x39) && // 0-9
            c !== 0x2e && // .
            c !== 0x2d && // -
            c !== 0x5f // _
        ) {
            // Less frequent chars are checked here
            // ":"
            if (c === 0x3a) {
                if (i === end - 1) {
                    throw new did_error_js_1.InvalidDidError(input, `DID cannot end with ":"`);
                }
                continue;
            }
            // pct-encoded
            if (c === 0x25) {
                c = input.charCodeAt(++i);
                if ((c < 0x30 || c > 0x39) && (c < 0x41 || c > 0x46)) {
                    throw new did_error_js_1.InvalidDidError(input, `Invalid pct-encoded character at position ${i}`);
                }
                c = input.charCodeAt(++i);
                if ((c < 0x30 || c > 0x39) && (c < 0x41 || c > 0x46)) {
                    throw new did_error_js_1.InvalidDidError(input, `Invalid pct-encoded character at position ${i}`);
                }
                // There must always be 2 HEXDIG after a "%"
                if (i >= end) {
                    throw new did_error_js_1.InvalidDidError(input, `Incomplete pct-encoded character at position ${i - 2}`);
                }
                continue;
            }
            throw new did_error_js_1.InvalidDidError(input, `Disallowed character in DID at position ${i}`);
        }
    }
}
function assertDid(input) {
    if (typeof input !== 'string') {
        throw new did_error_js_1.InvalidDidError(typeof input, `DID must be a string`);
    }
    const { length } = input;
    if (length > 2048) {
        throw new did_error_js_1.InvalidDidError(input, `DID is too long (2048 chars max)`);
    }
    if (!input.startsWith(DID_PREFIX)) {
        throw new did_error_js_1.InvalidDidError(input, `DID requires "${DID_PREFIX}" prefix`);
    }
    const idSep = input.indexOf(':', DID_PREFIX_LENGTH);
    if (idSep === -1) {
        throw new did_error_js_1.InvalidDidError(input, `Missing colon after method name`);
    }
    assertDidMethod(input, DID_PREFIX_LENGTH, idSep);
    assertDidMsid(input, idSep + 1, length);
}
function isDid(input) {
    try {
        assertDid(input);
        return true;
    }
    catch (err) {
        if (err instanceof did_error_js_1.DidError) {
            return false;
        }
        // Unexpected TypeError (should never happen)
        throw err;
    }
}
function asDid(input) {
    assertDid(input);
    return input;
}
exports.didSchema = zod_1.z
    .string()
    .superRefine((value, ctx) => {
    try {
        assertDid(value);
        return true;
    }
    catch (err) {
        ctx.addIssue({
            code: zod_1.z.ZodIssueCode.custom,
            message: err instanceof Error ? err.message : 'Unexpected error',
        });
        return false;
    }
});
//# sourceMappingURL=did.js.map