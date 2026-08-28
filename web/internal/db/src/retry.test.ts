import { afterEach, describe, expect, it, vi } from "vitest";
import type { Database, Transaction } from "./index";
import {
  DEFAULT_ATTEMPTS,
  databaseBackoffMs,
  retry,
  shouldRetryAfterCommit,
  shouldRetryBeforeCommit,
  transactionWithRetry,
} from "./retry";

/** The shape mysql2 throws. */
function driverError(fields: { code?: string; errno?: number }): Error {
  return Object.assign(new Error("mysql error"), fields);
}

const deadlock = () => driverError({ errno: 1213, code: "ER_LOCK_DEADLOCK" });
const lockWait = () => driverError({ errno: 1205, code: "ER_LOCK_WAIT_TIMEOUT" });
const connectionLost = () => driverError({ code: "PROTOCOL_CONNECTION_LOST" });
const duplicateKey = () => driverError({ errno: 1062, code: "ER_DUP_ENTRY" });

/** Records backoff durations instead of waiting for them. */
function recordingSleep() {
  const slept: number[] = [];
  return {
    slept,
    sleep: async (ms: number) => {
      slept.push(ms);
    },
  };
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("retry", () => {
  it("returns the result without retrying when the call succeeds", async () => {
    const fn = vi.fn().mockResolvedValue("ok");
    const { slept, sleep } = recordingSleep();

    await expect(retry(fn, { sleep })).resolves.toBe("ok");
    expect(fn).toHaveBeenCalledTimes(1);
    expect(slept).toEqual([]);
  });

  it("retries until the call succeeds", async () => {
    const fn = vi
      .fn()
      .mockRejectedValueOnce(new Error("first"))
      .mockRejectedValueOnce(new Error("second"))
      .mockResolvedValue("ok");

    await expect(retry(fn, { sleep: async () => {} })).resolves.toBe("ok");
    expect(fn).toHaveBeenCalledTimes(3);
  });

  it("throws the last error when every attempt fails", async () => {
    const fn = vi
      .fn()
      .mockRejectedValueOnce(new Error("first"))
      .mockRejectedValueOnce(new Error("second"))
      .mockRejectedValue(new Error("last"));

    await expect(retry(fn, { sleep: async () => {} })).rejects.toThrow("last");
    expect(fn).toHaveBeenCalledTimes(DEFAULT_ATTEMPTS);
  });

  it("stops on the first non retryable error", async () => {
    const fn = vi.fn().mockRejectedValue(duplicateKey());

    await expect(
      retry(fn, { shouldRetry: shouldRetryBeforeCommit, sleep: async () => {} }),
    ).rejects.toThrow("mysql error");
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("rejects a configuration with less than one attempt", async () => {
    const fn = vi.fn();

    await expect(retry(fn, { attempts: 0 })).rejects.toThrow("attempts must be an integer");
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects an attempt count that is not a number", async () => {
    const fn = vi.fn();

    await expect(retry(fn, { attempts: Number.NaN })).rejects.toThrow(
      "attempts must be an integer",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("does not run the call when the signal is already aborted", async () => {
    const fn = vi.fn();
    const controller = new AbortController();
    controller.abort();

    await expect(retry(fn, { signal: controller.signal })).rejects.toThrow();
    expect(fn).not.toHaveBeenCalled();
  });

  it("stops as soon as the signal aborts between attempts", async () => {
    const controller = new AbortController();
    const fn = vi.fn().mockImplementation(async () => {
      controller.abort();
      throw new Error("transient");
    });

    await expect(retry(fn, { signal: controller.signal })).rejects.toThrow();
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("does not retry an abort error raised inside the call", async () => {
    const fn = vi.fn().mockRejectedValue(new DOMException("aborted", "AbortError"));

    await expect(retry(fn, { sleep: async () => {} })).rejects.toThrow("aborted");
    expect(fn).toHaveBeenCalledTimes(1);
  });
});

describe("databaseBackoffMs", () => {
  /** Jitter keeps each value within half and one and a half times the base. */
  function expectAround(value: number, base: number) {
    expect(value).toBeGreaterThanOrEqual(base * 0.5);
    expect(value).toBeLessThanOrEqual(base * 1.5);
  }

  it("grows exponentially over the configured attempts", () => {
    expectAround(databaseBackoffMs(1), 50);
    expectAround(databaseBackoffMs(2), 100);
    expectAround(databaseBackoffMs(3), 200);
  });

  it("falls back to the base duration for an invalid attempt", () => {
    expectAround(databaseBackoffMs(0), 50);
  });
});

describe("shouldRetryBeforeCommit", () => {
  it("retries transient errors except a lock wait timeout", () => {
    expect(shouldRetryBeforeCommit(deadlock())).toBe(true);
    expect(shouldRetryBeforeCommit(connectionLost())).toBe(true);
    expect(shouldRetryBeforeCommit(lockWait())).toBe(false);
    expect(shouldRetryBeforeCommit(duplicateKey())).toBe(false);
    expect(shouldRetryBeforeCommit(new Error("application error"))).toBe(false);
  });
});

describe("shouldRetryAfterCommit", () => {
  it("retries only a deadlock", () => {
    expect(shouldRetryAfterCommit(deadlock())).toBe(true);
    expect(shouldRetryAfterCommit(lockWait())).toBe(false);
    expect(shouldRetryAfterCommit(connectionLost())).toBe(false);
    expect(shouldRetryAfterCommit(duplicateKey())).toBe(false);
  });
});

describe("transactionWithRetry", () => {
  /** Minimal drizzle stand-in: it only runs the transaction callback. */
  function fakeDatabase(): Pick<Database, "transaction"> {
    return {
      transaction: (fn) => fn({} as Transaction),
    };
  }

  it("runs the whole transaction again after a deadlock", async () => {
    const db = fakeDatabase();
    const work = vi.fn().mockRejectedValueOnce(deadlock()).mockResolvedValue("ok");

    await expect(transactionWithRetry(db, work, { sleep: async () => {} })).resolves.toBe("ok");
    expect(work).toHaveBeenCalledTimes(2);
  });

  it("retries a connection error raised before the callback ran", async () => {
    const work = vi.fn().mockResolvedValue("ok");
    let attempts = 0;
    const db: Pick<Database, "transaction"> = {
      transaction: async (fn) => {
        attempts++;
        if (attempts === 1) {
          throw connectionLost();
        }

        return fn({} as Transaction);
      },
    };

    await expect(transactionWithRetry(db, work, { sleep: async () => {} })).resolves.toBe("ok");
    expect(attempts).toBe(2);
    expect(work).toHaveBeenCalledTimes(1);
  });

  it("retries a connection error raised by a statement inside the callback", async () => {
    const db = fakeDatabase();
    const work = vi.fn().mockRejectedValueOnce(connectionLost()).mockResolvedValue("ok");

    await expect(transactionWithRetry(db, work, { sleep: async () => {} })).resolves.toBe("ok");
    expect(work).toHaveBeenCalledTimes(2);
  });

  it("does not re-run the callback after a connection error at commit, it may have landed", async () => {
    const work = vi.fn().mockResolvedValue("ok");
    const db: Pick<Database, "transaction"> = {
      transaction: async (fn) => {
        await fn({} as Transaction);
        throw connectionLost();
      },
    };

    await expect(transactionWithRetry(db, work, { sleep: async () => {} })).rejects.toThrow(
      "mysql error",
    );
    expect(work).toHaveBeenCalledTimes(1);
  });

  it("retries a deadlock at commit with exponential backoff", async () => {
    // Pins the jitter factor to 1.0 so the exact schedule can be asserted.
    vi.spyOn(Math, "random").mockReturnValue(0.5);
    const { slept, sleep } = recordingSleep();
    const work = vi.fn().mockResolvedValue("ok");
    let attempts = 0;
    const db: Pick<Database, "transaction"> = {
      transaction: async (fn) => {
        attempts++;
        const result = await fn({} as Transaction);
        if (attempts < 3) {
          throw deadlock();
        }
        return result;
      },
    };

    await expect(transactionWithRetry(db, work, { sleep })).resolves.toBe("ok");
    expect(attempts).toBe(3);
    expect(slept).toEqual([50, 100]);
  });

  it("does not retry a lock wait timeout", async () => {
    const db = fakeDatabase();
    const work = vi.fn().mockRejectedValue(lockWait());

    await expect(transactionWithRetry(db, work, { sleep: async () => {} })).rejects.toThrow(
      "mysql error",
    );
    expect(work).toHaveBeenCalledTimes(1);
  });

  it("does not retry an error thrown by the callback itself", async () => {
    const db = fakeDatabase();
    const work = vi.fn().mockRejectedValue(new Error("workos update failed"));

    await expect(transactionWithRetry(db, work, { sleep: async () => {} })).rejects.toThrow(
      "workos update failed",
    );
    expect(work).toHaveBeenCalledTimes(1);
  });
});
