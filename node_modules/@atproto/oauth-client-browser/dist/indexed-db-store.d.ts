import { SimpleStore, Key, Value } from '@atproto-labs/simple-store';
import { DBObjectStore } from './indexed-db/index.js';
type Item<V> = {
    value: V;
    createdAt: Date;
};
export declare class IndexedDBStore<K extends Extract<IDBValidKey, Key>, V extends Value> implements SimpleStore<K, V> {
    private dbName;
    protected maxAge: number;
    constructor(dbName: string, maxAge?: number);
    protected run<R>(mode: 'readonly' | 'readwrite', fn: (s: DBObjectStore<Item<V>>) => R | Promise<R>): Promise<R>;
    get(key: K): Promise<V | undefined>;
    set(key: K, value: V): Promise<void>;
    del(key: K): Promise<void>;
    deleteOutdated(): Promise<void>;
}
export {};
//# sourceMappingURL=indexed-db-store.d.ts.map