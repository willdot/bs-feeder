export type Simplify<T> = {
    [K in keyof T]: T[K];
} & NonNullable<unknown>;
export type TupleUnion<U extends string, R extends any[] = []> = {
    [S in U]: Exclude<U, S> extends never ? [...R, S] : TupleUnion<Exclude<U, S>, [...R, S]>;
}[U];
/**
 * @example
 * ```ts
 * const clientId = buildLoopbackClientId(window.location)
 * ```
 */
export declare function buildLoopbackClientId(location: {
    hostname: string;
    pathname: string;
    port: string;
}, localhost?: string): string;
interface TypedBroadcastChannelEventMap<T> {
    message: MessageEvent<T>;
    messageerror: MessageEvent<T>;
}
export interface TypedBroadcastChannel<T> extends EventTarget {
    readonly name: string;
    close(): void;
    postMessage(message: T): void;
    addEventListener<K extends keyof TypedBroadcastChannelEventMap<T>>(type: K, listener: (this: BroadcastChannel, ev: TypedBroadcastChannelEventMap<T>[K]) => any, options?: boolean | AddEventListenerOptions): void;
    addEventListener(type: string, listener: EventListenerOrEventListenerObject, options?: boolean | AddEventListenerOptions): void;
    removeEventListener<K extends keyof TypedBroadcastChannelEventMap<T>>(type: K, listener: (this: BroadcastChannel, ev: TypedBroadcastChannelEventMap<T>[K]) => any, options?: boolean | EventListenerOptions): void;
    removeEventListener(type: string, listener: EventListenerOrEventListenerObject, options?: boolean | EventListenerOptions): void;
}
export {};
//# sourceMappingURL=util.d.ts.map