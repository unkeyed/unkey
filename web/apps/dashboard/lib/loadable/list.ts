import { isCancelledError } from "@tanstack/react-query";
import { match } from "@unkey/match";
import type { ListFooterState } from "@unkey/ui";
import { useMemo, useState } from "react";
import { type LoadFailure, type Loadable, type QueryResult, fromQuery } from "./loadable";

export type ListView<Row> =
  | { status: "loading" }
  | ({ status: "failed" } & LoadFailure)
  | { status: "empty" }
  | { status: "ready"; rows: readonly Row[]; footer: ListFooterState };

type NoMatch = { status: "noMatch"; search: string };

export type SearchedListView<Row> = ListView<Row> | NoMatch;

type ShownList<Row> = Extract<ListView<Row>, { status: "empty" | "ready" }>;

export type NextPageAttempt =
  | { status: "idle" }
  | { status: "loading"; dataUpdatedAt: number }
  | { status: "failed"; error: unknown; dataUpdatedAt: number };

export type InfiniteQueryResult<Page> = QueryResult<{ pages: Page[] }> & {
  dataUpdatedAt: number;
  isPreviousData: boolean;
  hasNextPage?: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: (options: { throwOnError: boolean }) => Promise<unknown>;
};

const HIDDEN: ListFooterState = { kind: "hidden" };

const NO_ROWS: readonly never[] = [];

const NOTHING_SHOWN: ListTableState<never> = {
  rows: NO_ROWS,
  isLoading: false,
  footer: HIDDEN,
  failure: null,
};

export function listOf<Row>(rows: Loadable<readonly Row[]>): ListView<Row> {
  if (rows.status !== "ready") {
    return rows;
  }
  return toListView(rows.data, rows.stale ? refreshError(rows.stale) : HIDDEN);
}

export function infiniteListOf<Page, Row>({
  query,
  rows,
  attempt,
  loadMore,
}: {
  query: InfiniteQueryResult<Page>;
  rows: readonly Row[];
  attempt: NextPageAttempt;
  loadMore: () => void;
}): ListView<Row> {
  const pages = fromQuery(query);
  if (pages.status !== "ready") {
    return pages;
  }
  const list = toListView(rows, infiniteFooterOf(query, attempt, loadMore, pages.stale));
  if (list.status === "empty" && query.isPreviousData) {
    return { status: "loading" };
  }
  return list;
}

export function attemptAfterRejection(cause: unknown, dataUpdatedAt: number): NextPageAttempt {
  return isCancelledError(cause)
    ? { status: "idle" }
    : { status: "failed", error: cause, dataUpdatedAt };
}

function isNextPageLoading<Page>(query: InfiniteQueryResult<Page>, attempt: NextPageAttempt) {
  return (
    query.isFetchingNextPage ||
    (attempt.status === "loading" && attempt.dataUpdatedAt === query.dataUpdatedAt)
  );
}

function useNextPageAttempt<Page>(query: InfiniteQueryResult<Page>) {
  const [attempt, setAttempt] = useState<NextPageAttempt>({ status: "idle" });
  const loadMore = () => {
    if (isNextPageLoading(query, attempt)) {
      return;
    }
    const started: NextPageAttempt = { status: "loading", dataUpdatedAt: query.dataUpdatedAt };
    const settle = (next: NextPageAttempt) =>
      setAttempt((current) => (current === started ? next : current));
    setAttempt(started);
    query.fetchNextPage({ throwOnError: true }).then(
      () => settle({ status: "idle" }),
      (cause: unknown) => settle(attemptAfterRejection(cause, started.dataUpdatedAt)),
    );
  };
  return { attempt, loadMore };
}

export type ShownRows<Row> = {
  idOf?: (row: Row) => string;
  mapRows?: (rows: readonly Row[]) => readonly Row[];
};

export function uniqueRowsOf<Row>(rows: readonly Row[], idOf: (row: Row) => string): Row[] {
  const seen = new Set<string>();
  return rows.filter((row) => {
    const id = idOf(row);
    if (seen.has(id)) {
      return false;
    }
    seen.add(id);
    return true;
  });
}

export function useInfiniteList<Page, Row>(
  query: InfiniteQueryResult<Page>,
  rowsOf: (page: Page) => readonly Row[],
  { idOf, mapRows }: ShownRows<Row> = {},
): ListView<Row> {
  const { attempt, loadMore } = useNextPageAttempt(query);
  const pages = query.data?.pages;
  const rows = useMemo(() => {
    if (!pages) {
      return NO_ROWS;
    }
    const all = pages.flatMap(rowsOf);
    const unique = idOf ? uniqueRowsOf(all, idOf) : all;
    return mapRows ? mapRows(unique) : unique;
  }, [pages, rowsOf, idOf, mapRows]);
  return infiniteListOf({ query, rows, attempt, loadMore });
}

type ListTableState<Row> = {
  rows: readonly Row[];
  isLoading: boolean;
  footer: ListFooterState;
  failure: LoadFailure | null;
};

export function tableStateOf<Row>(view: ListView<Row>): ListTableState<Row> {
  return match(view)
    .with({ status: "loading" }, () => ({ ...NOTHING_SHOWN, isLoading: true }))
    .with({ status: "failed" }, ({ error, retry, retrying }) => ({
      ...NOTHING_SHOWN,
      failure: { error, retry, retrying },
    }))
    .with({ status: "empty" }, () => NOTHING_SHOWN)
    .with({ status: "ready" }, ({ rows, footer }) => ({
      rows,
      isLoading: false,
      footer,
      failure:
        rows.length === 0 && footer.kind === "error"
          ? { error: footer.error, retry: footer.onRetry, retrying: footer.retrying }
          : null,
    }))
    .exhaustive();
}

export function shownTotalOf({
  loaded,
  total,
  hasMore,
}: {
  loaded: number;
  total: number | null;
  hasMore: boolean;
}): number | null {
  if (!hasMore) {
    return loaded;
  }
  return total === null ? null : Math.max(total, loaded);
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
  const loadMore = () => {
    setShown({ forSearch: search, limit: limit + pageSize });
  };
  return { limit, loadMore };
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
  stale: LoadFailure | null,
): ListFooterState {
  if (query.isPreviousData) {
    return HIDDEN;
  }
  const loadMoreError = (cause: unknown): ListFooterState => ({
    kind: "error",
    reason: "loadMore",
    error: cause,
    onRetry: loadMore,
    retrying: false,
  });
  if (attempt.status === "failed" && attempt.dataUpdatedAt === query.dataUpdatedAt) {
    return loadMoreError(attempt.error);
  }
  const nextPageLoading = isNextPageLoading(query, attempt);
  // v4 sets isError before fetchNextPage rejects; without this a failed load-more flashes as a refresh error.
  if (nextPageLoading && !query.isFetchingNextPage && query.isError) {
    return loadMoreError(query.error);
  }
  if (nextPageLoading) {
    return { kind: "loadingMore" };
  }
  if (stale) {
    return refreshError(stale);
  }
  return query.hasNextPage ? { kind: "more", onLoadMore: loadMore } : HIDDEN;
}

function refreshError({ error, retry, retrying }: LoadFailure): ListFooterState {
  return { kind: "error", reason: "refresh", error, onRetry: retry, retrying };
}

function toListView<Row>(rows: readonly Row[], footer: ListFooterState): ShownList<Row> {
  if (rows.length === 0 && footer.kind === "hidden") {
    return { status: "empty" };
  }
  return { status: "ready", rows, footer };
}
