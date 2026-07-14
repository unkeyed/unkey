import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { serializeFilters } from "@/hooks/serialize-transition-key";
import { usePageChange } from "@/hooks/use-page-change";
import { usePageClamp } from "@/hooks/use-page-clamp";
import { usePageTransition } from "@/hooks/use-page-transition";
import { usePrefetchPages } from "@/hooks/use-prefetch-pages";
import { trpc } from "@/lib/trpc/client";
import { parseAsInteger, useQueryState } from "nuqs";
import { useMemo } from "react";
import { type AuditLogsQueryPayload, DEFAULT_BUCKET_NAME } from "../schema/audit-logs.schema";

const DEFAULT_PAGE_SIZE = 50;

export function useAuditLogsQuery(pageSize = DEFAULT_PAGE_SIZE) {
  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  const filtersKey = useMemo(() => serializeFilters(filters), [filters]);

  const queryPage = usePageTransition({
    transitionKey: filtersKey,
    page: normalizedPage,
    setPage,
  });

  const queryParams = useMemo(() => {
    const params: AuditLogsQueryPayload = {
      limit: pageSize,
      page: queryPage,
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
  }, [filters, pageSize, queryPage]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.audit.logs.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));

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
      utils.audit.logs.prefetch(params, { staleTime: Number.POSITIVE_INFINITY }),
  });

  const onPageChange = usePageChange(totalPages, setPage);

  return {
    auditLogs: data?.auditLogs ?? [],
    isLoading,
    isFetching,
    page: queryPage,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
  };
}
