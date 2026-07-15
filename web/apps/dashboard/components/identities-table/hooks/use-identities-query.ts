import { useFilters } from "@/app/(app)/[workspaceSlug]/identities/hooks/use-filters";
import { parseAsSortArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { serializeFilters } from "@/hooks/serialize-transition-key";
import { usePageChange } from "@/hooks/use-page-change";
import { usePageClamp } from "@/hooks/use-page-clamp";
import { usePageTransition } from "@/hooks/use-page-transition";
import { usePrefetchPages } from "@/hooks/use-prefetch-pages";
import { trpc } from "@/lib/trpc/client";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, parseAsString, useQueryState } from "nuqs";
import { useCallback, useMemo } from "react";
import type { IdentitiesFilterOperator, IdentitiesSortField } from "../schema/identities.schema";

const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 100;

// Bidirectional mapping between TanStack column IDs and server sort field names
const COLUMN_ID_TO_SORT_FIELD: Record<string, IdentitiesSortField> = {
  externalId: "externalId",
  created: "createdAt",
  keys: "keyCount",
  ratelimits: "ratelimitCount",
  last_used: "lastUsed",
};
const SORT_FIELD_TO_COLUMN_ID: Record<IdentitiesSortField, string> = {
  externalId: "externalId",
  createdAt: "created",
  keyCount: "keys",
  ratelimitCount: "ratelimits",
  lastUsed: "last_used",
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

  const { filters } = useFilters();

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

  // Filters and search both change the result set, so either changing
  // invalidates the current page.
  const filtersKey = useMemo(
    () => `${serializeFilters(filters)}|search:${JSON.stringify(search)}`,
    [filters, search],
  );

  const queryPage = usePageTransition({
    transitionKey: filtersKey,
    page: normalizedPage,
    setPage,
  });

  const filterParams = useMemo(() => {
    const externalId = filters
      .filter((f) => f.field === "externalId")
      .map((f) => ({ operator: f.operator as IdentitiesFilterOperator, value: f.value as string }));
    const lastUsedSince = filters.find((f) => f.field === "lastUsedSince")?.value as
      | string
      | undefined;
    const lastUsedStart = filters.find((f) => f.field === "lastUsedStart")?.value as
      | number
      | undefined;
    const lastUsedEnd = filters.find((f) => f.field === "lastUsedEnd")?.value as number | undefined;
    return {
      externalId: externalId.length > 0 ? externalId : undefined,
      lastUsedSince,
      lastUsedStart,
      lastUsedEnd,
    };
  }, [filters]);

  const queryParams = useMemo(
    () => ({
      ...filterParams,
      page: queryPage,
      limit: normalizedPageSize,
      search: search || undefined,
      sortBy: sortParams?.[0]?.column ?? "createdAt",
      sortOrder: sortParams?.[0]?.direction ?? "desc",
    }),
    [filterParams, queryPage, normalizedPageSize, search, sortParams],
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

  usePageClamp({
    page: queryPage,
    totalPages,
    data,
    setPage,
  });

  usePrefetchPages({
    page: queryPage,
    totalPages,
    queryParams,
    prefetch: (params) =>
      utils.identity.query.prefetch(params, { staleTime: Number.POSITIVE_INFINITY }),
  });

  const onPageChange = usePageChange(totalPages, setPage);

  return {
    identities: data?.identities ?? [],
    isLoading,
    isInitialLoading,
    isFetching,
    page: queryPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
