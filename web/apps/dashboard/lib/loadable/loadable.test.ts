import { describe, expect, test } from "vitest";
import { type QueryResult, fromQuery } from "./loadable";

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

  test("no data with an error and a refetch in flight is failed and retrying", () => {
    const error = new Error("boom");
    expect(fromQuery(query({ isError: true, isFetching: true, error }))).toEqual({
      status: "failed",
      error,
      retry: expect.any(Function),
      retrying: true,
    });
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
