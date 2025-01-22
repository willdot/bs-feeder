import { Json } from '@atproto-labs/fetch';
export declare class OAuthResponseError extends Error {
    readonly response: Response;
    readonly payload: Json;
    readonly error?: string;
    readonly errorDescription?: string;
    constructor(response: Response, payload: Json);
    get status(): number;
    get headers(): Headers;
}
//# sourceMappingURL=oauth-response-error.d.ts.map