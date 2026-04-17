import { parseAsSortArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { trpc } from "@/lib/trpc/client";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, parseAsString, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import type { IdentitiesSortField } from "../schema/identities.schema";

const PREFETCH_PAGES_AHEAD = 2;
const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 100;

// Bidirectional mapping between TanStack column IDs and server sort field names
const COLUMN_ID_TO_SORT_FIELD: Record<string, IdentitiesSortField> = {
  externalId: "externalId",
  created: "createdAt",
  keys: "keyCount",
  ratelimits: "ratelimitCount",
};
const SORT_FIELD_TO_COLUMN_ID: Record<IdentitiesSortField, string> = {
  externalId: "externalId",
  createdAt: "created",
  keyCount: "keys",
  ratelimitCount: "ratelimits",
};

export function useIdentitiesQuery(pageSize = DEFAULT_PAGE_SIZE) {
  const normalizedPageSize =
    Number.isFinite(pageSize) && pageSize > 0
      ? Math.min(Math.floor(pageSize), MAX_PAGE_SIZE)
      : DEFAULT_PAGE_SIZE;

  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  const [search] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );

  const [sortParams, setSortParams] = useQueryState(
    "sort",
    parseAsSortArray<IdentitiesSortField>(),
  );

  // Convert URL sort params → TanStack SortingState
  const sorting: SortingState = useMemo(() => {
    if (!sortParams || sortParams.length === 0) {
      return [{ id: "created", desc: true }];
    }
    return sortParams.map((s) => ({
      id: SORT_FIELD_TO_COLUMN_ID[s.column] ?? s.column,
      desc: s.direction === "desc",
    }));
  }, [sortParams]);

  // Convert TanStack SortingState → URL sort params; reset to page 1
  const onSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      setSortParams(
        next.length === 0
          ? null
          : next
              .filter((s) => COLUMN_ID_TO_SORT_FIELD[s.id] !== undefined)
              .map((s) => ({
                column: COLUMN_ID_TO_SORT_FIELD[s.id],
                direction: s.desc ? "desc" : "asc",
              })),
      );
      setPage(1);
    },
    [sorting, setSortParams, setPage],
  );

  // Reset to page 1 only when search actually changes (not on initial mount).
  const prevSearchRef = useRef<string | null>(null);
  useEffect(() => {
    if (prevSearchRef.current === null) {
      prevSearchRef.current = search;
      return;
    }
    if (search !== prevSearchRef.current) {
      prevSearchRef.current = search;
      setPage(1);
    }
  }, [search, setPage]);

  const queryParams = useMemo(
    () => ({
      page: normalizedPage,
      limit: normalizedPageSize,
      search: search || undefined,
      sortBy: sortParams?.[0]?.column ?? "createdAt",
      sortOrder: sortParams?.[0]?.direction ?? "desc",
    }),
    [normalizedPage, normalizedPageSize, search, sortParams],
  );

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.identity.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const isInitialLoading = isLoading && !data;

  const totalCount = data?.total ?? 0;
  const totalPages = data?.totalPages ?? 1;

  // Clamp page to valid range after data/totalPages updates.
  useEffect(() => {
    if (normalizedPage > totalPages) {
      setPage(totalPages);
    }
  }, [normalizedPage, totalPages, setPage]);

  // Prefetch the next few pages so navigation feels instant.
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = normalizedPage + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.identity.query.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [normalizedPage, totalPages, queryParams, utils.identity.query]);

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );

  return {
    identities: data?.identities ?? [],
    isLoading: isInitialLoading,
    isFetching,
    page: normalizedPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
