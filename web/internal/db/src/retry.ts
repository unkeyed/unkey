import {
  isDeadlockError,
  isDuplicateKeyError,
  isLockWaitTimeoutError,
  isTransientError,
} from "./errors";
import type { Database, Transaction } from "./index";

/**
 * Retry helpers for database work, the TypeScript counterpart of `pkg/retry`
 * and `pkg/db/retry.go`. `AbortSignal` takes the place of `context.Context`.
 */

export const DEFAULT_BACKOFF_MS = 50;
export const DEFAULT_ATTEMPTS = 3;

export type RetryOptions = {
  /** Maximum number of attempts, the first one included. */
  attempts?: number;
  /** Milliseconds to wait before the next attempt. `attempt` starts at 1. */
  backoffMs?: (attempt: number) => number;
  /** When omitted, every error is retried. */
  shouldRetry?: (err: unknown) => boolean;
  /** Aborts between attempts and during the backoff sleep. */
  signal?: AbortSignal;
  /** Test seam, so tests do not wait in real time. */
  sleep?: (ms: number, signal?: AbortSignal) => Promise<void>;
};

function sleepMs(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(signal.reason);
      return;
    }

    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, ms);

    function onAbort() {
      clearTimeout(timer);
      reject(signal?.reason);
    }

    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

function isAbortError(err: unknown): boolean {
  return err instanceof Error && (err.name === "AbortError" || err.name === "TimeoutError");
}

/**
 * Runs `fn` until it succeeds, the attempts run out, or the error is not
 * retryable. Rethrows the error of the last attempt.
 */
export async function retry<T>(fn: () => Promise<T>, options: RetryOptions = {}): Promise<T> {
  const attempts = options.attempts ?? DEFAULT_ATTEMPTS;
  if (attempts < 1) {
    throw new Error("attempts must be greater than 0");
  }

  const backoffMs = options.backoffMs ?? (() => DEFAULT_BACKOFF_MS);
  const sleep = options.sleep ?? sleepMs;
  const { signal, shouldRetry } = options;

  let lastError: unknown;

  for (let attempt = 1; attempt <= attempts; attempt++) {
    signal?.throwIfAborted();

    try {
      return await fn();
    } catch (err) {
      lastError = err;

      // A signal aborted inside `fn` ends the loop, even when `signal` is still alive.
      if (isAbortError(err)) {
        throw err;
      }

      if (shouldRetry && !shouldRetry(err)) {
        throw err;
      }

      if (attempt < attempts) {
        await sleep(backoffMs(attempt), signal);
      }
    }
  }

  throw lastError;
}

/** Doubles per attempt: 50ms, 100ms, 200ms. With 3 attempts only the first two are used. */
export function databaseBackoffMs(attempt: number): number {
  if (attempt < 1) {
    return DEFAULT_BACKOFF_MS;
  }

  return DEFAULT_BACKOFF_MS * 2 ** (attempt - 1);
}

/**
 * Retries transient errors only. Duplicate keys are permanent, and application
 * errors thrown inside a transaction are left alone.
 *
 * Unlike the Go version there is no "not found" case: drizzle returns an empty
 * result set instead of throwing when a row is missing.
 */
export function shouldRetryDatabaseError(err: unknown): boolean {
  if (isDuplicateKeyError(err)) {
    return false;
  }

  return isTransientError(err);
}

/**
 * Narrower than `shouldRetryDatabaseError`. A deadlock or lock wait timeout
 * is safe to re-run: InnoDB rolls back the whole transaction on a deadlock and
 * only the statement on a lock wait timeout, but drizzle issues `ROLLBACK`
 * before rethrowing either way, so the next attempt starts from `BEGIN` on a
 * clean connection. A connection error is not safe once the callback has run:
 * the commit may already have landed, and a re-run would write the callback's
 * side effects twice.
 */
export function shouldRetryTransactionError(err: unknown): boolean {
  return isDeadlockError(err) || isLockWaitTimeoutError(err);
}

function databaseRetryOptions(
  shouldRetry: (err: unknown) => boolean,
  options: Pick<RetryOptions, "signal" | "sleep">,
): RetryOptions {
  return {
    attempts: DEFAULT_ATTEMPTS,
    backoffMs: databaseBackoffMs,
    shouldRetry,
    signal: options.signal,
    sleep: options.sleep,
  };
}

/** Runs a database operation with 3 attempts and a 50/100ms backoff. */
export function withRetry<T>(
  fn: () => Promise<T>,
  options: Pick<RetryOptions, "signal" | "sleep"> = {},
): Promise<T> {
  return retry(fn, databaseRetryOptions(shouldRetryDatabaseError, options));
}

/**
 * Runs a transaction with 3 attempts and a 50/100ms backoff. The whole
 * transaction restarts from `BEGIN`, so the callback may not depend on state
 * from a rolled back attempt.
 *
 * Driver errors must reach this loop unwrapped or wrapped with `cause`. An
 * inner `.catch` that throws a bare `TRPCError` hides the MySQL error code and
 * silently disables the retry.
 *
 * drizzle acquires the connection and runs `BEGIN` before the callback. An
 * error there cannot have committed anything, so the wider transient set is
 * safe to retry until the callback has started.
 */
export function transactionWithRetry<T>(
  db: Pick<Database, "transaction">,
  fn: (tx: Transaction) => Promise<T>,
  options: Pick<RetryOptions, "signal" | "sleep"> = {},
): Promise<T> {
  let callbackStarted = false;

  return retry(
    () => {
      callbackStarted = false;
      return db.transaction((tx) => {
        callbackStarted = true;
        return fn(tx);
      });
    },
    databaseRetryOptions(
      (err) => (callbackStarted ? shouldRetryTransactionError(err) : shouldRetryDatabaseError(err)),
      options,
    ),
  );
}
