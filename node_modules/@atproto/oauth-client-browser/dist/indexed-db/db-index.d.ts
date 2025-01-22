import { ObjectStoreSchema } from './schema.js';
export declare class DBIndex<Schema extends ObjectStoreSchema> {
    private idbIndex;
    constructor(idbIndex: IDBIndex);
    count(query?: IDBValidKey | IDBKeyRange): Promise<number>;
    get(query: IDBValidKey | IDBKeyRange): Promise<Schema>;
    getKey(query: IDBValidKey | IDBKeyRange): Promise<IDBValidKey | undefined>;
    getAll(query?: IDBValidKey | IDBKeyRange | null, count?: number): Promise<Schema[]>;
    getAllKeys(query?: IDBValidKey | IDBKeyRange | null, count?: number): Promise<IDBValidKey[]>;
    deleteAll(query?: IDBValidKey | IDBKeyRange | null): Promise<void>;
}
//# sourceMappingURL=db-index.d.ts.map