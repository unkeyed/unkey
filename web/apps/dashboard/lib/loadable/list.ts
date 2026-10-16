import { isCancelledError } from "@tanstack/react-query";
import { useState } from "react";
import { type Failure, type Loadable, type QueryResult, fromQuery } from "./loadable";

export type ListFooterState =
  | { kind: "hidden" }
  | { kind: "more"; onLoadMore: () => void }
  | { kind: "loadingMore" }
  | {
      kind: "error";
      reason: "refresh" | "loadMore";
      error: unknown;
      onRetry: () => void;
      retrying: boolean;
    };

export type ListView<Row> =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | { status: "empty" }
  | { status: "ready"; rows: readonly Row[]; footer: ListFooterState };

export type NextPageAttempt =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "failed"; error: unknown; dataUpdatedAt: number };

export type InfiniteQueryResult<Page> = QueryResult<{ pages: Page[] }> & {
  dataUpdatedAt: number;
  hasNextPage?: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: (options: { throwOnError: boolean }) => Promise<unknown>;
};

const HIDDEN: ListFooterState = { kind: "hidden" };

export function listOf<Row>(rows: Loadable<readonly Row[]>): ListView<Row> {
  if (rows.status !== "ready") {
    return rows;
  }
  return toListView(rows.data, rows.stale ? refreshError(rows.stale) : HIDDEN);
}

export function infiniteListOf<Page, Row>({
  query,
  rowsOf,
  attempt,
  loadMore,
}: {
  query: InfiniteQueryResult<Page>;
  rowsOf: (page: Page) => readonly Row[];
  attempt: NextPageAttempt;
  loadMore: () => void;
}): ListView<Row> {
  const pages = fromQuery(query);
  if (pages.status !== "ready") {
    return pages;
  }
  return toListView(
    pages.data.pages.flatMap(rowsOf),
    infiniteFooterOf(query, attempt, loadMore, pages.stale),
  );
}

export function attemptAfterRejection(error: unknown, dataUpdatedAt: number): NextPageAttempt {
  return isCancelledError(error) ? { status: "idle" } : { status: "failed", error, dataUpdatedAt };
}

export function useInfiniteList<Page, Row>(
  query: InfiniteQueryResult<Page>,
  rowsOf: (page: Page) => readonly Row[],
): ListView<Row> {
  const [attempt, setAttempt] = useState<NextPageAttempt>({ status: "idle" });
  const loadMore = () => {
    const { dataUpdatedAt } = query;
    setAttempt({ status: "loading" });
    query.fetchNextPage({ throwOnError: true }).then(
      () => setAttempt({ status: "idle" }),
      (error: unknown) => setAttempt(attemptAfterRejection(error, dataUpdatedAt)),
    );
  };
  return infiniteListOf({ query, rowsOf, attempt, loadMore });
}

function infiniteFooterOf<Page>(
  query: InfiniteQueryResult<Page>,
  attempt: NextPageAttempt,
  loadMore: () => void,
  stale: Failure | null,
): ListFooterState {
  if (attempt.status === "failed" && attempt.dataUpdatedAt === query.dataUpdatedAt) {
    return {
      kind: "error",
      reason: "loadMore",
      error: attempt.error,
      onRetry: loadMore,
      retrying: false,
    };
  }
  // v4 sets isError before fetchNextPage rejects; without this a failed load-more flashes as a refresh error.
  if (attempt.status === "loading" && !query.isFetchingNextPage && query.isError) {
    return {
      kind: "error",
      reason: "loadMore",
      error: query.error,
      onRetry: loadMore,
      retrying: false,
    };
  }
  if (query.isFetchingNextPage) {
    return { kind: "loadingMore" };
  }
  if (stale) {
    return refreshError(stale);
  }
  return query.hasNextPage ? { kind: "more", onLoadMore: loadMore } : HIDDEN;
}

function refreshError({ error, retry, retrying }: Failure): ListFooterState {
  return { kind: "error", reason: "refresh", error, onRetry: retry, retrying };
}

function toListView<Row>(rows: readonly Row[], footer: ListFooterState): ListView<Row> {
  if (rows.length === 0 && footer.kind === "hidden") {
    return { status: "empty" };
  }
  return { status: "ready", rows, footer };
}
