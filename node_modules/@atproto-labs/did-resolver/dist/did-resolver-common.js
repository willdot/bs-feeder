"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DidResolverCommon = void 0;
const did_resolver_base_js_1 = require("./did-resolver-base.js");
const plc_js_1 = require("./methods/plc.js");
const web_js_1 = require("./methods/web.js");
class DidResolverCommon extends did_resolver_base_js_1.DidResolverBase {
    constructor(options) {
        super({
            plc: new plc_js_1.DidPlcMethod(options),
            web: new web_js_1.DidWebMethod(options),
        });
    }
}
exports.DidResolverCommon = DidResolverCommon;
//# sourceMappingURL=did-resolver-common.js.map