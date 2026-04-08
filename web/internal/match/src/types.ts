import type { PatternMatcher } from "./patterns";

/**
 * Resolves handler return type.
 * When `TConstraint` is set (via `.returnType<T>()`), handlers must return `TConstraint`.
 * When `TConstraint` is `never` (default), handler return type is inferred freely.
 */
export type HandlerReturn<TConstraint, TResult> = [TConstraint] extends [never]
  ? TResult
  : TConstraint;

// ── Narrowing ───────────────────────────────────────────────────────────────

/**
 * Selects which union variant(s) of `TInput` match `TPattern`.
 *
 * For `PatternMatcher<TNarrow>`, uses `Extract` against the phantom type.
 * For object patterns, checks all keys recursively.
 * For literals, checks direct assignability.
 */
export type NarrowByPattern<TInput, TPattern> = TInput extends unknown
  ? TPattern extends PatternMatcher<infer TNarrow>
    ? Extract<TInput, TNarrow> extends never
      ? [TNarrow] extends [TInput]
        ? TInput
        : never
      : Extract<TInput, TNarrow>
    : TPattern extends Record<string, unknown>
      ? TInput extends Record<string, unknown>
        ? NarrowObject<TInput, TPattern>
        : never
      : TPattern extends TInput
        ? TPattern
        : never
  : never;

/**
 * For each key in TInput, if the pattern constrains it, narrow that prop;
 * otherwise keep TInput's prop. If any pattern key narrows to `never` (or is
 * absent from TInput), the whole match fails and returns `never`.
 */
type NarrowObject<
  TInput extends Record<string, unknown>,
  TPattern extends Record<string, unknown>,
> = true extends {
  [K in keyof TPattern]: K extends keyof TInput
    ? [NarrowByPattern<TInput[K], TPattern[K]>] extends [never]
      ? true
      : false
    : true;
}[keyof TPattern]
  ? never
  : {
      [K in keyof TInput]: K extends keyof TPattern
        ? NarrowByPattern<TInput[K], TPattern[K]>
        : TInput[K];
    };

/**
 * All shapes accepted by `.with()` against an input of type `T`.
 *
 * - `T`                       — literal/value of the input type itself
 * - `PatternMatcher<T>`       — narrowing matchers like `P.string`
 * - `PatternMatcher<unknown>` — wildcard matchers like `P._`
 * - tuple/object branches     — recursive partial patterns over each key
 */
export type Pattern<T> =
  | T
  | PatternMatcher<T>
  | PatternMatcher<unknown>
  | (T extends readonly unknown[]
      ? { readonly [K in keyof T]: Pattern<T[K]> }
      : T extends object
        ? { readonly [K in keyof T]?: Pattern<T[K]> }
        : never);
