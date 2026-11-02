import { describe, expect, test } from "vitest";
import { failed, loading, ready } from "./fixtures";
import { type QueryResult, fromQuery, mapLoadable, readyLoadable, zipLoadable } from "./loadable";

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

  test("no data with an error and a refetch in flight fails retrying", () => {
    const error = new Error("boom");
    expect(fromQuery(query({ isError: true, isFetching: true, error }))).toEqual({
      status: "failed",
      error,
      retry: expect.any(Function),
      retrying: true,
    });
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

describe("mapLoadable", () => {
  test("maps ready data and passes other states through", () => {
    expect(mapLoadable({ status: "ready", data: 2, stale: null }, (n) => n * 2)).toEqual({
      status: "ready",
      data: 4,
      stale: null,
    });
    expect(mapLoadable({ status: "loading" }, (n: number) => n * 2)).toEqual({
      status: "loading",
    });
  });
});

describe("readyLoadable", () => {
  test("wraps a value with no stale failure", () => {
    expect(readyLoadable(null)).toEqual({ status: "ready", data: null, stale: null });
  });
});

describe("zipLoadable", () => {
  const sum = (a: number, b: number) => a + b;

  test("combines two ready values and keeps the first stale failure", () => {
    expect(zipLoadable(ready(1), ready(2), sum)).toEqual({ status: "ready", data: 3, stale: null });
    const stale = failed();
    expect(zipLoadable(ready(1), { ...ready(2), stale }, sum)).toEqual({
      status: "ready",
      data: 3,
      stale,
    });
  });

  test("waits for both and reports the first failure", () => {
    expect(zipLoadable(ready(1), loading, sum)).toEqual({ status: "loading" });
    const failure = failed();
    expect(zipLoadable(failure, loading, sum)).toBe(failure);
    expect(zipLoadable(ready(1), failure, sum)).toBe(failure);
  });

  test("retries every failed side", () => {
    const retries: string[] = [];
    const failing = (name: string) => ({ ...failed(), retry: () => retries.push(name) });
    const both = zipLoadable(failing("a"), failing("b"), sum);
    if (both.status === "failed") {
      both.retry();
    }
    const stale = zipLoadable(
      { ...ready(1), stale: failing("c") },
      { ...ready(2), stale: failing("d") },
      sum,
    );
    if (stale.status === "ready") {
      stale.stale?.retry();
    }
    expect(retries).toEqual(["a", "b", "c", "d"]);
  });
});
