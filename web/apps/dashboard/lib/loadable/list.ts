import { isCancelledError } from "@tanstack/react-query";
import { useState } from "react";
import { type Failure, type Loadable, type QueryResult, type Stale, fromQuery } from "./loadable";

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

type NoMatch = { status: "noMatch"; search: string };

export type SearchedListView<Row> = ListView<Row> | NoMatch;

type ShownList<Row> = Extract<ListView<Row>, { status: "empty" | "ready" }>;

export type NextPageAttempt =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "failed"; error: unknown; dataUpdatedAt: number };

export type InfiniteQueryResult<Page> = QueryResult<{ pages: Page[] }> & {
  dataUpdatedAt: number;
  isPreviousData: boolean;
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
  const list = toListView(
    pages.data.pages.flatMap(rowsOf),
    infiniteFooterOf(query, attempt, loadMore, pages.stale),
  );
  if (list.status === "empty" && query.isPreviousData) {
    return { status: "loading" };
  }
  return list;
}

export function attemptAfterRejection(error: unknown, dataUpdatedAt: number): NextPageAttempt {
  return isCancelledError(error) ? { status: "idle" } : { status: "failed", error, dataUpdatedAt };
}

function useNextPageAttempt<Page>(query: InfiniteQueryResult<Page>) {
  const [attempt, setAttempt] = useState<NextPageAttempt>({ status: "idle" });
  const loadMore = () => {
    const { dataUpdatedAt } = query;
    setAttempt({ status: "loading" });
    query.fetchNextPage({ throwOnError: true }).then(
      () => setAttempt({ status: "idle" }),
      (error: unknown) => setAttempt(attemptAfterRejection(error, dataUpdatedAt)),
    );
  };
  return { attempt, loadMore };
}

export function useInfiniteList<Page, Row>(
  query: InfiniteQueryResult<Page>,
  rowsOf: (page: Page) => readonly Row[],
): ListView<Row> {
  const { attempt, loadMore } = useNextPageAttempt(query);
  return infiniteListOf({ query, rowsOf, attempt, loadMore });
}

export function searchedListOf<View extends { status: string }>(
  list: View,
  search: string,
): View | NoMatch {
  if (list.status === "empty" && search !== "") {
    return { status: "noMatch", search };
  }
  return list;
}

export function usePageLimit(search: string, pageSize: number) {
  const [shown, setShown] = useState({ forSearch: search, limit: pageSize });
  const limit = shown.forSearch === search ? shown.limit : pageSize;
  return { limit, loadMore: () => setShown({ forSearch: search, limit: limit + pageSize }) };
}

export function shownListOf<Row>({
  all,
  matches,
  search,
  limit,
  loadMore,
}: {
  all: readonly Row[];
  matches: (rows: readonly Row[]) => readonly Row[];
  search: string;
  limit: number;
  loadMore: () => void;
}): SearchedListView<Row> {
  if (all.length === 0) {
    return { status: "empty" };
  }
  const rows = matches(all);
  const footer: ListFooterState =
    rows.length > limit ? { kind: "more", onLoadMore: loadMore } : HIDDEN;
  return searchedListOf(toListView(rows.slice(0, limit), footer), search);
}

export function useShowMore<Row>({
  all,
  isLoading,
  matches,
  search,
  pageSize,
}: {
  all: readonly Row[];
  isLoading: boolean;
  matches: (rows: readonly Row[]) => readonly Row[];
  search: string;
  pageSize: number;
}): SearchedListView<Row> {
  const { limit, loadMore } = usePageLimit(search, pageSize);
  if (isLoading) {
    return { status: "loading" };
  }
  return shownListOf({ all, matches, search, limit, loadMore });
}

export function moreFooterOf({
  hasMore,
  isLoadingMore = false,
  loadMore,
}: {
  hasMore: boolean | undefined;
  isLoadingMore?: boolean;
  loadMore: () => void;
}): ListFooterState {
  if (isLoadingMore) {
    return { kind: "loadingMore" };
  }
  return hasMore ? { kind: "more", onLoadMore: loadMore } : HIDDEN;
}

function infiniteFooterOf<Page>(
  query: InfiniteQueryResult<Page>,
  attempt: NextPageAttempt,
  loadMore: () => void,
  stale: Stale | null,
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
  if (query.isPreviousData || !query.hasNextPage) {
    return HIDDEN;
  }
  return { kind: "more", onLoadMore: loadMore };
}

function refreshError({ error, retry, retrying }: Stale): ListFooterState {
  return { kind: "error", reason: "refresh", error, onRetry: retry, retrying };
}

function toListView<Row>(rows: readonly Row[], footer: ListFooterState): ShownList<Row> {
  if (rows.length === 0 && footer.kind === "hidden") {
    return { status: "empty" };
  }
  return { status: "ready", rows, footer };
}
