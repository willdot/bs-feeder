export declare function handleRequest<T>(request: IDBRequest<T>, onSuccess: (result: T) => void, onError: (error: Error) => void): void;
export declare function promisify<T>(request: IDBRequest<T>): Promise<T>;
//# sourceMappingURL=util.d.ts.map