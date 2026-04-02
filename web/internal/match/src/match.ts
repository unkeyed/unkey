import { NonExhaustiveError } from "./errors";
import { matchPattern } from "./patterns";
import type { DeepPattern, HandlerReturn, NarrowByPattern } from "./types";

type MatchState<TOutput> = { matched: true; value: TOutput } | { matched: false; value: undefined };

const unmatched: MatchState<never> = {
  matched: false,
  value: undefined,
};

/**
 * Builder for a pattern matching expression. Chain `.with()` / `.when()` arms
 * and terminate with `.exhaustive()`, `.otherwise()`, or `.run()`.
 *
 * **Rule of thumb:** Use `.with()` for discriminated unions and null/type checks.
 * Use `.when()` for property-level predicates on non-discriminated types.
 */
class MatchBuilder<TInput, TOutput, TRemaining, TConstraint = never> {
  private constructor(
    private input: TInput,
    private state: MatchState<TOutput>,
  ) {}

  static create<T>(input: T): MatchBuilder<T, never, T> {
    return new MatchBuilder(input, unmatched);
  }

  private cast<O = TOutput, R = TRemaining>(): MatchBuilder<TInput, O, R, TConstraint> {
    return this as never;
  }

  /** Match a single pattern. */
  with<const TPattern, TResult>(
    pattern: TPattern & DeepPattern<TRemaining, NoInfer<TPattern>>,
    handler: (value: NarrowByPattern<TRemaining, TPattern>) => HandlerReturn<TConstraint, TResult>,
  ): MatchBuilder<
    TInput,
    TOutput | HandlerReturn<TConstraint, TResult>,
    Exclude<TRemaining, NarrowByPattern<TRemaining, TPattern>>,
    TConstraint
  >;

  /** Match a pattern with a guard predicate. */
  with<const TPattern, TResult>(
    pattern: TPattern & DeepPattern<TRemaining, NoInfer<TPattern>>,
    guard: (value: NarrowByPattern<TRemaining, TPattern>) => boolean,
    handler: (value: NarrowByPattern<TRemaining, TPattern>) => HandlerReturn<TConstraint, TResult>,
  ): MatchBuilder<TInput, TOutput | HandlerReturn<TConstraint, TResult>, TRemaining, TConstraint>;

  /** Match two patterns with OR semantics. */
  with<const P1, const P2, TResult>(
    p1: P1 & DeepPattern<TRemaining, NoInfer<P1>>,
    p2: P2 & DeepPattern<TRemaining, NoInfer<P2>>,
    handler: (
      value: NarrowByPattern<TRemaining, P1> | NarrowByPattern<TRemaining, P2>,
    ) => HandlerReturn<TConstraint, TResult>,
  ): MatchBuilder<
    TInput,
    TOutput | HandlerReturn<TConstraint, TResult>,
    Exclude<TRemaining, NarrowByPattern<TRemaining, P1> | NarrowByPattern<TRemaining, P2>>,
    TConstraint
  >;

  // biome-ignore lint: implementation signature must be loose to cover all overloads
  with(...args: any[]): any {
    if (this.state.matched) {
      return this.cast();
    }

    const handler = args[args.length - 1] as (value: unknown) => unknown;
    const patterns: unknown[] = [args[0]];
    let predicate: ((value: unknown) => unknown) | undefined = undefined;

    if (args.length === 3 && typeof args[1] === "function") {
      predicate = args[1] as (value: unknown) => unknown;
    } else if (args.length > 2) {
      patterns.push(...args.slice(1, args.length - 1));
    }

    const matched =
      patterns.some((pattern) => matchPattern(pattern, this.input)) &&
      (predicate ? Boolean(predicate(this.input)) : true);

    const state: MatchState<unknown> = matched
      ? { matched: true, value: handler(this.input) }
      : unmatched;

    return new MatchBuilder(this.input, state).cast();
  }

  /** Match using a type guard predicate — narrows `TRemaining`. */
  when<TNarrowed extends TInput, TResult>(
    predicate: (value: TInput) => value is TNarrowed,
    handler: (value: TNarrowed) => HandlerReturn<TConstraint, TResult>,
  ): MatchBuilder<
    TInput,
    TOutput | HandlerReturn<TConstraint, TResult>,
    Exclude<TRemaining, TNarrowed>,
    TConstraint
  >;

  /** Match using a boolean predicate — does not narrow. */
  when<TResult>(
    predicate: (value: TInput) => boolean,
    handler: (value: TInput) => HandlerReturn<TConstraint, TResult>,
  ): MatchBuilder<TInput, TOutput | HandlerReturn<TConstraint, TResult>, TRemaining, TConstraint>;

  // biome-ignore lint: implementation signature must be loose to cover all overloads
  when(predicate: (value: any) => boolean, handler: (value: any) => any): any {
    if (this.state.matched) {
      return this.cast();
    }

    const matched = Boolean(predicate(this.input));

    return new MatchBuilder(
      this.input,
      matched ? { matched: true, value: handler(this.input) } : unmatched,
    ).cast();
  }

  /** Provide a fallback handler for all remaining (unmatched) cases. */
  otherwise<TResult>(
    handler: (value: TRemaining) => HandlerReturn<TConstraint, TResult>,
  ): TOutput | HandlerReturn<TConstraint, TResult> {
    if (this.state.matched) {
      return this.state.value;
    }
    return handler(this.input as never);
  }

  /**
   * Terminate the match expression. Compile-time error if not all cases are covered.
   * Throws `NonExhaustiveError` at runtime if called when a case is missing.
   */
  exhaustive(..._args: [TRemaining] extends [never] ? [] : [never]): TOutput {
    if (this.state.matched) {
      return this.state.value;
    }
    throw new NonExhaustiveError(this.input);
  }

  /** Alias for `.exhaustive()`. */
  run(...args: [TRemaining] extends [never] ? [] : [never]): TOutput {
    return this.exhaustive(...args);
  }

  /**
   * Constrain all handler return types to `TNewOutput`.
   * Eliminates `as const` on object literals via contextual typing.
   */
  returnType<TNewOutput>(): MatchBuilder<TInput, never, TRemaining, TNewOutput> {
    return this.cast<never, TRemaining>() as never;
  }

  /** Type-only no-op. Returns `this` unchanged. */
  narrow(): MatchBuilder<TInput, TOutput, TRemaining, TConstraint> {
    return this;
  }
}

/**
 * Create a pattern matching expression.
 *
 * @example
 * ```ts
 * match(status)
 *   .with("ready", () => <Ready />)
 *   .with("error", () => <Error />)
 *   .exhaustive();
 * ```
 */
export function match<const T>(value: T): MatchBuilder<T, never, T> {
  return MatchBuilder.create(value);
}
