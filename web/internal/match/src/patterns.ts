const MATCHER = Symbol("@unkey/match/matcher");

/**
 * A pattern helper object. Carries a runtime `match` function via the
 * `[MATCHER]` symbol and a phantom `_narrow` type for compile-time narrowing.
 */
export interface PatternMatcher<TNarrow = unknown> {
  readonly [MATCHER]: { match(value: unknown): boolean };
  readonly _narrow: TNarrow;
}

export function isMatcher(x: unknown): x is PatternMatcher {
  return typeof x === "object" && x !== null && MATCHER in x;
}

function createMatcher<TNarrow>(matchFn: (value: unknown) => boolean): PatternMatcher<TNarrow> {
  return { [MATCHER]: { match: matchFn }, _narrow: undefined as never };
}

const isObject = (value: unknown): value is Record<string, unknown> =>
  Boolean(value && typeof value === "object");

/** Recursively checks if `value` matches `pattern`. Uses `Object.is` for primitives. */
export function matchPattern(pattern: unknown, value: unknown): boolean {
  if (isMatcher(pattern)) {
    return pattern[MATCHER].match(value);
  }

  if (!isObject(pattern)) {
    return Object.is(value, pattern);
  }

  if (!isObject(value)) return false;

  if (Array.isArray(pattern)) {
    if (!Array.isArray(value)) return false;
    return (
      pattern.length === value.length &&
      pattern.every((subPattern, i) => matchPattern(subPattern, value[i]))
    );
  }

  return Reflect.ownKeys(pattern).every((k) => {
    const subPattern = (pattern as Record<string | symbol, unknown>)[k];
    return k in value && matchPattern(subPattern, (value as Record<string | symbol, unknown>)[k]);
  });
}

const any: PatternMatcher<unknown> = createMatcher(() => true);

/**
 * Pattern helpers for use inside `.with()`.
 *
 * **Rule of thumb:** Use `.with()` for discriminated unions and null/type checks.
 * Use `.when()` for property-level predicates on non-discriminated types.
 */
export const P = {
  /** Wildcard — matches any value */
  _: any,
  /** Wildcard — matches any value */
  any,
  /** Matches any string */
  string: createMatcher<string>((v) => typeof v === "string"),
  /** Matches any number */
  number: createMatcher<number>((v) => typeof v === "number"),
  /** Matches any boolean */
  boolean: createMatcher<boolean>((v) => typeof v === "boolean"),
  /** Matches null or undefined */
  nullish: createMatcher<null | undefined>((v) => v == null),
  /** Matches any value that is not null or undefined */
  nonNullable: createMatcher<{}>((v) => v != null),
  /** Matches using an inline predicate function */
  when<TInput>(predicate: (value: TInput) => boolean): PatternMatcher<TInput> {
    return createMatcher((v) => predicate(v as TInput));
  },
} as const;
