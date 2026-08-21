/**
 * Database error helpers for the dashboard's drizzle + mysql2 stack.
 */

import { DrizzleQueryError } from "@unkey/db";

/**
 * Returns the driver error a DrizzleQueryError wraps, or undefined when it
 * carries none. Kept next to [isDuplicateKeyError] so drizzle's wrapper
 * shape is encoded in this file only; the walk is bounded in case a future
 * drizzle nests its wrapper.
 */
export function unwrapDrizzleQueryError(err: DrizzleQueryError): unknown {
  let cause: unknown = err.cause;
  for (let depth = 0; cause instanceof DrizzleQueryError && depth < 10; depth++) {
    cause = cause.cause;
  }
  return cause;
}

/**
 * Detects MySQL duplicate-key violations (ER_DUP_ENTRY / errno 1062).
 *
 * drizzle-orm (>= 0.36) wraps every driver error in a `DrizzleQueryError` and
 * stores the original mysql2 error on `.cause`, so the fields we care about are
 * nested rather than on the top-level error. We walk the cause chain to find
 * them, which keeps the check correct regardless of how many layers wrap the
 * driver error.
 */
export function isDuplicateKeyError(err: unknown): boolean {
  let current: unknown = err;

  // Bound the walk to guard against accidental cycles in the cause chain.
  for (let depth = 0; current != null && depth < 10; depth++) {
    if (typeof current === "object") {
      const candidate = current as { code?: unknown; errno?: unknown; cause?: unknown };
      if (candidate.code === "ER_DUP_ENTRY" || candidate.errno === 1062) {
        return true;
      }
      current = candidate.cause;
    } else {
      break;
    }
  }

  return false;
}
