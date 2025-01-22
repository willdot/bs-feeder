import { DidResolverBase } from './did-resolver-base.js';
import { DidPlcMethodOptions } from './methods/plc.js';
import { DidWebMethodOptions } from './methods/web.js';
import { Simplify } from './util.js';
export type DidResolverCommonOptions = Simplify<DidPlcMethodOptions & DidWebMethodOptions>;
export declare class DidResolverCommon extends DidResolverBase<'plc' | 'web'> implements DidResolverBase<'plc' | 'web'> {
    constructor(options?: DidResolverCommonOptions);
}
//# sourceMappingURL=did-resolver-common.d.ts.map