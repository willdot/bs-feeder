import { DBObjectStore } from './db-object-store.js';
import { DatabaseSchema } from './schema.js';
export declare class DBTransaction<Schema extends DatabaseSchema> implements Disposable {
    #private;
    constructor(tx: IDBTransaction);
    protected get tx(): IDBTransaction;
    abort(): Promise<void>;
    commit(): Promise<void>;
    objectStore<T extends keyof Schema & string>(name: T): DBObjectStore<Schema[T]>;
    [Symbol.dispose](): void;
}
//# sourceMappingURL=db-transaction.d.ts.map