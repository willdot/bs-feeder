import { DBIndex } from './db-index.js';
import { ObjectStoreSchema } from './schema.js';
export declare class DBObjectStore<Schema extends ObjectStoreSchema> {
    private idbObjStore;
    constructor(idbObjStore: IDBObjectStore);
    get name(): string;
    index(name: string): DBIndex<Schema>;
    get(key: IDBValidKey | IDBKeyRange): Promise<Schema>;
    getKey(query: IDBValidKey | IDBKeyRange): Promise<IDBValidKey | undefined>;
    getAll(query?: IDBValidKey | IDBKeyRange | null, count?: number): Promise<Schema[]>;
    getAllKeys(query?: IDBValidKey | IDBKeyRange | null, count?: number): Promise<IDBValidKey[]>;
    add(value: Schema, key?: IDBValidKey): Promise<IDBValidKey>;
    put(value: Schema, key?: IDBValidKey): Promise<IDBValidKey>;
    delete(key: IDBValidKey | IDBKeyRange): Promise<undefined>;
    clear(): Promise<undefined>;
}
//# sourceMappingURL=db-object-store.d.ts.map