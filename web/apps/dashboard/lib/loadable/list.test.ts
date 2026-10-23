import { CancelledError } from "@tanstack/react-query";
import { describe, expect, test } from "vitest";
import {
  type InfiniteQueryResult,
  type ListView,
  type NextPageAttempt,
  attemptAfterRejection,
  infiniteListOf,
  listOf,
} from "./list";
import { type QueryResult, fromQuery } from "./loadable";

type Page = { rows: string[] };

function recorder() {
  const calls: string[] = [];
  return {
    calls,
    refetch: async () => {
      calls.push("refetch");
    },
    loadMore: () => {
      calls.push("loadMore");
    },
  };
}

function infinite(
  overrides: Partial<InfiniteQueryResult<Page>>,
  calls: ReturnType<typeof recorder>,
): InfiniteQueryResult<Page> {
  return {
    data: undefined,
    error: null,
    isError: false,
    isFetching: false,
    refetch: calls.refetch,
    dataUpdatedAt: 100,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: async () => undefined,
    ...overrides,
  };
}

function view(
  overrides: Partial<InfiniteQueryResult<Page>>,
  attempt: NextPageAttempt = { status: "idle" },
  calls = recorder(),
): ListView<string> {
  return infiniteListOf({
    query: infinite(overrides, calls),
    rowsOf: (page) => page.rows,
    attempt,
    loadMore: calls.loadMore,
  });
}

function pressFooter(result: ListView<string>) {
  if (result.status !== "ready") {
    throw new Error(`expected ready, got ${result.status}`);
  }
  const { footer } = result;
  if (footer.kind === "more") {
    footer.onLoadMore();
  }
  if (footer.kind === "error") {
    footer.onRetry();
  }
}

const pages = (...rows: string[][]) => ({ pages: rows.map((r) => ({ rows: r })) });
const error = new Error("boom");

describe("listOf", () => {
  const calls = recorder();

  test("rows with no error are ready with no footer", () => {
    expect(
      listOf(
        fromQuery({
          data: ["a"],
          error: null,
          isError: false,
          isFetching: false,
          refetch: calls.refetch,
        }),
      ),
    ).toEqual({
      status: "ready",
      rows: ["a"],
      footer: { kind: "hidden" },
    });
  });

  test("a refresh error keeps rows and offers a refetch", () => {
    const local = recorder();
    const result = listOf(
      fromQuery({ data: ["a"], error, isError: true, isFetching: false, refetch: local.refetch }),
    );
    expect(result).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(local.calls).toEqual(["refetch"]);
  });

  test("zero rows is empty", () => {
    expect(
      listOf(
        fromQuery({
          data: [],
          error: null,
          isError: false,
          isFetching: false,
          refetch: calls.refetch,
        }),
      ),
    ).toEqual({
      status: "empty",
    });
  });

  test("zero rows with a refresh error stays ready so the error shows", () => {
    const local = recorder();
    const result = listOf(
      fromQuery({ data: [], error, isError: true, isFetching: false, refetch: local.refetch }),
    );
    expect(result).toEqual({
      status: "ready",
      rows: [],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(local.calls).toEqual(["refetch"]);
  });

  test("no data with an error fails and carries the error", () => {
    expect(
      listOf(
        fromQuery<QueryResult<string[]>>({
          data: undefined,
          error,
          isError: true,
          isFetching: false,
          refetch: calls.refetch,
        }),
      ),
    ).toEqual({ status: "failed", error, retry: expect.any(Function) });
  });

  test("no data and no error is loading", () => {
    expect(
      listOf(
        fromQuery<QueryResult<string[]>>({
          data: undefined,
          error: null,
          isError: false,
          isFetching: false,
          refetch: calls.refetch,
        }),
      ),
    ).toEqual({ status: "loading" });
  });
});

describe("infiniteListOf", () => {
  test("flattens rows across pages", () => {
    expect(view({ data: pages(["a", "b"], ["c"]) })).toEqual({
      status: "ready",
      rows: ["a", "b", "c"],
      footer: { kind: "hidden" },
    });
  });

  test("no data and no error is loading", () => {
    expect(view({})).toEqual({ status: "loading" });
  });

  test("no data with an error fails and retries by refetching", () => {
    const calls = recorder();
    const result = view({ isError: true, error }, { status: "idle" }, calls);
    expect(result).toEqual({
      status: "failed",
      error,
      retry: expect.any(Function),
    });
    if (result.status === "failed") {
      result.retry();
    }
    expect(calls.calls).toEqual(["refetch"]);
  });

  test("zero rows and no more pages is empty", () => {
    expect(view({ data: pages([]) })).toEqual({ status: "empty" });
  });

  test("zero rows with a refresh error and no more pages stays ready so the error shows", () => {
    const calls = recorder();
    const result = view({ data: pages([]), isError: true, error }, { status: "idle" }, calls);
    expect(result).toEqual({
      status: "ready",
      rows: [],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(calls.calls).toEqual(["refetch"]);
  });

  test("zero rows with more pages stays ready so the user can load more", () => {
    expect(view({ data: pages([]), hasNextPage: true })).toEqual({
      status: "ready",
      rows: [],
      footer: { kind: "more", onLoadMore: expect.any(Function) },
    });
  });

  test("more pages offers load more", () => {
    const calls = recorder();
    const result = view({ data: pages(["a"]), hasNextPage: true }, { status: "idle" }, calls);
    expect(result).toEqual({
      status: "ready",
      rows: ["a"],
      footer: { kind: "more", onLoadMore: expect.any(Function) },
    });
    pressFooter(result);
    expect(calls.calls).toEqual(["loadMore"]);
  });

  test("fetching the next page shows loading more", () => {
    expect(
      view(
        { data: pages(["a"]), hasNextPage: true, isFetchingNextPage: true },
        { status: "loading" },
      ),
    ).toEqual({ status: "ready", rows: ["a"], footer: { kind: "loadingMore" } });
  });

  test("a refresh error keeps rows and retries by refetching", () => {
    const calls = recorder();
    const result = view({ data: pages(["a"]), isError: true, error }, { status: "idle" }, calls);
    expect(result).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(calls.calls).toEqual(["refetch"]);
  });

  test("a refresh error beats load more", () => {
    const calls = recorder();
    const result = view(
      { data: pages(["a"]), hasNextPage: true, isError: true, error },
      { status: "idle" },
      calls,
    );
    expect(result).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(calls.calls).toEqual(["refetch"]);
  });

  test("a recorded next-page failure beats a refresh error and retries by loading more", () => {
    const calls = recorder();
    const pageError = new Error("page");
    const result = view(
      { data: pages(["a"]), hasNextPage: true, isError: true, error, dataUpdatedAt: 100 },
      { status: "failed", error: pageError, dataUpdatedAt: 100 },
      calls,
    );
    expect(result).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "loadMore",
        error: pageError,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(calls.calls).toEqual(["loadMore"]);
  });

  test("an in-flight next page that errors is a next-page failure, not a refresh failure", () => {
    const calls = recorder();
    const result = view(
      { data: pages(["a"]), hasNextPage: true, isError: true, error },
      { status: "loading" },
      calls,
    );
    expect(result).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "loadMore",
        error,
        onRetry: expect.any(Function),
        retrying: false,
      },
    });
    pressFooter(result);
    expect(calls.calls).toEqual(["loadMore"]);
  });

  test("retrying a next page while the query still holds the old error shows loading more", () => {
    expect(
      view(
        { data: pages(["a"]), hasNextPage: true, isFetchingNextPage: true, isError: true, error },
        { status: "loading" },
      ),
    ).toEqual({ status: "ready", rows: ["a"], footer: { kind: "loadingMore" } });
  });

  test("a later successful refetch clears a recorded next-page failure", () => {
    expect(
      view(
        { data: pages(["a"]), hasNextPage: true, dataUpdatedAt: 200 },
        { status: "failed", error, dataUpdatedAt: 100 },
      ),
    ).toEqual({
      status: "ready",
      rows: ["a"],
      footer: { kind: "more", onLoadMore: expect.any(Function) },
    });
  });
});

describe("retrying", () => {
  test("a refresh error with a refetch in flight shows the refresh footer retrying", () => {
    expect(view({ data: pages(["a"]), isError: true, isFetching: true, error })).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: true,
      },
    });
  });

  test("a stale list query with a refetch in flight shows the refresh footer retrying", () => {
    const calls = recorder();
    expect(
      listOf(
        fromQuery({ data: ["a"], error, isError: true, isFetching: true, refetch: calls.refetch }),
      ),
    ).toEqual({
      status: "ready",
      rows: ["a"],
      footer: {
        kind: "error",
        reason: "refresh",
        error,
        onRetry: expect.any(Function),
        retrying: true,
      },
    });
  });
});

describe("attemptAfterRejection", () => {
  test("a cancelled next-page load returns to idle", () => {
    expect(attemptAfterRejection(new CancelledError(), 100)).toEqual({ status: "idle" });
  });

  test("any other rejection is a failure stamped with the data version", () => {
    expect(attemptAfterRejection(error, 100)).toEqual({
      status: "failed",
      error,
      dataUpdatedAt: 100,
    });
  });
});
