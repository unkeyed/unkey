import { describe, expect, test } from "vitest";
import { type Loadable, type QueryResult, blockerOf, fromQuery } from "./loadable";

function query<T>(overrides: Partial<QueryResult<T>>, refetches: string[] = []): QueryResult<T> {
  return {
    data: undefined,
    error: null,
    isError: false,
    isFetching: false,
    refetch: async () => {
      refetches.push("refetch");
    },
    ...overrides,
  };
}

describe("fromQuery", () => {
  test("no data and no error is loading", () => {
    expect(fromQuery(query({}))).toEqual({ status: "loading" });
  });

  test("no data with an error fails and retries by refetching", () => {
    const refetches: string[] = [];
    const error = new Error("boom");
    const result = fromQuery(query({ isError: true, error }, refetches));

    expect(result).toEqual({
      status: "failed",
      error,
      retry: expect.any(Function),
      retrying: false,
    });
    if (result.status === "failed") {
      result.retry();
    }
    expect(refetches).toEqual(["refetch"]);
  });

  test("data without an error is ready and fresh", () => {
    expect(fromQuery(query({ data: ["a"] }))).toEqual({
      status: "ready",
      data: ["a"],
      stale: null,
    });
  });

  test("data with an error stays ready and marks it stale", () => {
    const refetches: string[] = [];
    const error = new Error("refresh failed");
    const result = fromQuery(query({ data: ["a"], isError: true, error }, refetches));

    expect(result).toEqual({
      status: "ready",
      data: ["a"],
      stale: { error, retry: expect.any(Function), retrying: false },
    });
    if (result.status === "ready") {
      result.stale?.retry();
    }
    expect(refetches).toEqual(["refetch"]);
  });

  test("data with an error and a refetch in flight marks the stale failure retrying", () => {
    const error = new Error("refresh failed");
    expect(fromQuery(query({ data: ["a"], isError: true, isFetching: true, error }))).toEqual({
      status: "ready",
      data: ["a"],
      stale: { error, retry: expect.any(Function), retrying: true },
    });
  });
});

describe("blockerOf", () => {
  const ready: Loadable<string[]> = { status: "ready", data: ["a"], stale: null };

  test("every view ready is nothing pending", () => {
    expect(blockerOf([ready, ready])).toBeNull();
  });

  test("no views is nothing pending", () => {
    expect(blockerOf([])).toBeNull();
  });

  test("a ready view with a failed refresh does not block", () => {
    const stale = { error: new Error("refresh"), retry: () => {}, retrying: false };
    expect(blockerOf([{ status: "ready", data: ["a"], stale }])).toBeNull();
  });

  test("one view loading is loading", () => {
    expect(blockerOf([ready, { status: "loading" }])).toEqual({ status: "loading" });
  });

  test("a failed view outranks a loading one and retries every failed view", () => {
    const retries: string[] = [];
    const first = new Error("domains");
    const result = blockerOf([
      { status: "loading" },
      { status: "failed", error: first, retrying: false, retry: () => retries.push("domains") },
      ready,
      {
        status: "failed",
        error: new Error("custom"),
        retrying: false,
        retry: () => retries.push("custom"),
      },
    ]);

    expect(result).toEqual({
      status: "failed",
      error: first,
      retrying: false,
      retry: expect.any(Function),
    });
    if (result?.status === "failed") {
      result.retry();
    }
    expect(retries).toEqual(["domains", "custom"]);
  });
});
