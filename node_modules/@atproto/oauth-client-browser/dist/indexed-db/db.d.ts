import { DatabaseSchema } from './schema.js';
import { DBTransaction } from './db-transaction.js';
export declare class DB<Schema extends DatabaseSchema> implements Disposable {
    #private;
    protected readonly txOptions?: IDBTransactionOptions | undefined;
    static open<Schema extends DatabaseSchema = DatabaseSchema>(dbName: string, migrations: ReadonlyArray<(db: IDBDatabase) => void>, txOptions?: IDBTransactionOptions): Promise<DB<Schema>>;
    constructor(db: IDBDatabase, txOptions?: IDBTransactionOptions | undefined);
    protected get db(): IDBDatabase;
    get name(): string;
    get objectStoreNames(): DOMStringList;
    get version(): number;
    transaction<T extends readonly (keyof Schema & string)[], R>(storeNames: T, mode: IDBTransactionMode, run: (tx: DBTransaction<Pick<Schema, T[number]>>) => R | PromiseLike<R>): Promise<R>;
    close(): void;
    [Symbol.dispose](): void;
}
//# sourceMappingURL=db.d.ts.map