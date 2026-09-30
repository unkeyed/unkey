import { DrizzleQueryError } from "drizzle-orm/errors";
import { describe, expect, it } from "vitest";
import {
  isConnectionError,
  isDeadlockError,
  isDuplicateKeyError,
  isLockWaitTimeoutError,
  isTooManyConnectionsError,
  isTransientError,
} from "./errors";

/** The shape mysql2 throws. */
function driverError(fields: { code?: string; errno?: number; message?: string }): Error {
  return Object.assign(new Error(fields.message ?? "mysql error"), {
    code: fields.code,
    errno: fields.errno,
  });
}

/** Wraps an error the way drizzle wraps driver errors. */
function wrapped(cause: Error, depth = 1): Error {
  let current = cause;
  for (let i = 0; i < depth; i++) {
    current = new DrizzleQueryError("select 1", [], current);
  }

  return current;
}

describe("isDeadlockError", () => {
  it("matches errno 1213", () => {
    expect(isDeadlockError(driverError({ errno: 1213, code: "ER_LOCK_DEADLOCK" }))).toBe(true);
  });

  it("matches through nested causes", () => {
    expect(isDeadlockError(wrapped(driverError({ errno: 1213 }), 3))).toBe(true);
  });

  it("does not match other errors", () => {
    expect(isDeadlockError(driverError({ errno: 1062, code: "ER_DUP_ENTRY" }))).toBe(false);
  });
});

describe("isLockWaitTimeoutError", () => {
  it("matches errno 1205", () => {
    expect(isLockWaitTimeoutError(driverError({ errno: 1205 }))).toBe(true);
  });
});

describe("isTooManyConnectionsError", () => {
  it("matches errno 1040", () => {
    expect(isTooManyConnectionsError(driverError({ errno: 1040 }))).toBe(true);
  });
});

describe("isConnectionError", () => {
  it("matches server gone and server lost", () => {
    expect(isConnectionError(driverError({ errno: 2006 }))).toBe(true);
    expect(isConnectionError(driverError({ errno: 2013 }))).toBe(true);
  });

  it("matches socket level errors that carry no mysql error number", () => {
    expect(isConnectionError(driverError({ code: "ECONNRESET" }))).toBe(true);
    expect(isConnectionError(driverError({ code: "PROTOCOL_CONNECTION_LOST" }))).toBe(true);
  });

  it("does not match a query error", () => {
    expect(isConnectionError(driverError({ errno: 1146, code: "ER_NO_SUCH_TABLE" }))).toBe(false);
  });
});

describe("isDuplicateKeyError", () => {
  it("matches errno 1062 and ER_DUP_ENTRY", () => {
    expect(isDuplicateKeyError(driverError({ errno: 1062 }))).toBe(true);
    expect(isDuplicateKeyError(driverError({ code: "ER_DUP_ENTRY" }))).toBe(true);
  });

  it("matches through nested causes", () => {
    expect(isDuplicateKeyError(wrapped(driverError({ errno: 1062 }), 2))).toBe(true);
  });
});

describe("isTransientError", () => {
  it("is true for deadlocks, lock waits, connections and connection counts", () => {
    expect(isTransientError(driverError({ errno: 1213 }))).toBe(true);
    expect(isTransientError(driverError({ errno: 1205 }))).toBe(true);
    expect(isTransientError(driverError({ errno: 1040 }))).toBe(true);
    expect(isTransientError(driverError({ code: "ECONNREFUSED" }))).toBe(true);
  });

  it("is false for permanent and non database errors", () => {
    expect(isTransientError(driverError({ errno: 1062 }))).toBe(false);
    expect(isTransientError(new Error("boom"))).toBe(false);
    expect(isTransientError(null)).toBe(false);
    expect(isTransientError("ER_LOCK_DEADLOCK")).toBe(false);
  });

  it("stops walking a cyclic cause chain", () => {
    const first: { cause?: unknown } = {};
    const second = { cause: first };
    first.cause = second;

    expect(isTransientError(first)).toBe(false);
  });
});
