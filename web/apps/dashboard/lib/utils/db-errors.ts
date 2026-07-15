/**
 * Database error helpers for the dashboard's drizzle + mysql2 stack.
 */

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
