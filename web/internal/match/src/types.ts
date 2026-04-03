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

/** Returns `TInput` if ALL pattern keys can match, `never` otherwise. */
type NarrowObject<
  TInput extends Record<string, unknown>,
  TPattern extends Record<string, unknown>,
> = false extends {
  [K in keyof TPattern]: K extends keyof TInput
    ? [NarrowByPattern<TInput[K], TPattern[K]>] extends [never]
      ? false
      : true
    : false;
}[keyof TPattern]
  ? never
  : TInput;

/**
 * Validates that a pattern is structurally compatible with the input type.
 * `PatternMatcher` is always valid; objects are checked key-by-key.
 */
export type DeepPattern<TInput, TPattern> = TPattern extends PatternMatcher
  ? TPattern
  : TPattern extends Record<string, unknown>
    ? TInput extends Record<string, unknown>
      ? {
          readonly [K in keyof TPattern]: K extends keyof TInput
            ? DeepPattern<TInput[K], TPattern[K]>
            : TPattern[K];
        }
      : TPattern
    : TPattern;
