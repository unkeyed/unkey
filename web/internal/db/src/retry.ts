import { setTimeout as sleepMs } from "node:timers/promises";
import { isDeadlockError, isLockWaitTimeoutError, isTransientError } from "./errors";
import type { Database, Transaction } from "./index";

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

function isAbortError(err: unknown): boolean {
  return err instanceof Error && (err.name === "AbortError" || err.name === "TimeoutError");
}

/** Rethrows the error of the last attempt. */
export async function retry<T>(fn: () => Promise<T>, options: RetryOptions = {}): Promise<T> {
  const attempts = options.attempts ?? DEFAULT_ATTEMPTS;
  if (!Number.isInteger(attempts) || attempts < 1) {
    throw new Error("attempts must be an integer greater than 0");
  }

  const backoffMs = options.backoffMs ?? (() => DEFAULT_BACKOFF_MS);
  const sleep = options.sleep ?? ((ms, signal) => sleepMs(ms, undefined, { signal }));
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

/** Doubles per attempt with 0.5x to 1.5x jitter, so requests that failed together do not retry together. */
export function databaseBackoffMs(attempt: number): number {
  const base = attempt < 1 ? DEFAULT_BACKOFF_MS : DEFAULT_BACKOFF_MS * 2 ** (attempt - 1);
  return Math.round(base * (0.5 + Math.random()));
}

/** A lock wait timeout is excluded: the lock holder is still there after the backoff, so a retry waits the full 50s again. */
export function shouldRetryBeforeCommit(err: unknown): boolean {
  return isTransientError(err) && !isLockWaitTimeoutError(err);
}

/**
 * A deadlock cannot come from `COMMIT`, so nothing has landed. A connection
 * error can, so a re-run could write twice.
 */
export function shouldRetryAfterCommit(err: unknown): boolean {
  return isDeadlockError(err);
}

/**
 * Restarts from `BEGIN`, so the callback may not keep state across attempts.
 * Driver errors must reach this loop unwrapped or with `cause` set, or the
 * retry is silently disabled.
 */
export function transactionWithRetry<T>(
  db: Pick<Database, "transaction">,
  fn: (tx: Transaction) => Promise<T>,
  options: Pick<RetryOptions, "signal" | "sleep"> = {},
): Promise<T> {
  let commitStarted = false;

  return retry(
    () => {
      commitStarted = false;
      return db.transaction(async (tx) => {
        const result = await fn(tx);
        commitStarted = true;
        return result;
      });
    },
    {
      attempts: DEFAULT_ATTEMPTS,
      backoffMs: databaseBackoffMs,
      shouldRetry: (err) =>
        commitStarted ? shouldRetryAfterCommit(err) : shouldRetryBeforeCommit(err),
      signal: options.signal,
      sleep: options.sleep,
    },
  );
}
