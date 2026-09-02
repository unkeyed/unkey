import { useMemo } from "react";
import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import {
  computeTotalPages,
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  paginationFilterKey,
  usePaginatedNavigation,
  usePaginatedPage,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import { type AuditLogsQueryPayload, DEFAULT_BUCKET_NAME } from "../schema/audit-logs.schema";

const DEFAULT_PAGE_SIZE = 50;

// Audit logs are paginated but not sortable, so this composes the shared
// pagination primitives directly rather than usePaginatedListQuery (which owns
// a URL `sort` param). The primitives own page state, the deep-link clamp, and
// prefetch; the filter-to-payload mapping stays here.
export function useAuditLogsQuery(pageSize = DEFAULT_PAGE_SIZE) {
  const { filters } = useFilters();

  const filtersKey = useMemo(() => paginationFilterKey(filters), [filters]);

  const { page, setPage } = usePaginatedPage(filtersKey);

  const queryParams = useMemo(() => {
    const params: AuditLogsQueryPayload = {
      limit: pageSize,
      page,
      startTime: undefined,
      endTime: undefined,
      events: { filters: [] },
      users: { filters: [] },
      rootKeys: { filters: [] },
      since: "",
      bucket: DEFAULT_BUCKET_NAME,
    };

    for (const filter of filters) {
      switch (filter.field) {
        case "events": {
          if (typeof filter.value === "string") {
            params.events?.filters.push({ operator: filter.operator, value: filter.value });
          }
          break;
        }
        case "rootKeys": {
          if (typeof filter.value === "string") {
            params.rootKeys?.filters.push({ operator: filter.operator, value: filter.value });
          }
          break;
        }
        case "users": {
          if (typeof filter.value === "string") {
            params.users?.filters.push({ operator: filter.operator, value: filter.value });
          }
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value === "number") {
            params[filter.field] = filter.value;
          }
          break;
        }
        case "since": {
          if (typeof filter.value === "string") {
            params.since = filter.value;
          }
          break;
        }
        case "bucket": {
          if (typeof filter.value === "string") {
            params.bucket = filter.value;
          }
          break;
        }
      }
    }

    return params;
  }, [filters, pageSize, page]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.audit.logs.useQuery(
    queryParams,
    PAGINATED_LIST_QUERY_OPTIONS,
  );

  const totalCount = data?.total ?? 0;
  const totalPages = computeTotalPages(totalCount, pageSize);

  const { onPageChange, isInitialLoading, isNavigating } = usePaginatedNavigation({
    data,
    page,
    totalPages,
    setPage,
    isLoading,
    isFetching,
    queryParams,
    prefetch: (params) => utils.audit.logs.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
  });

  return {
    auditLogs: data?.auditLogs ?? [],
    isLoading: isInitialLoading,
    isNavigating,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
  };
}
