import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { trpc } from "@/lib/trpc/client";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { type AuditLogsQueryPayload, DEFAULT_BUCKET_NAME } from "../schema/audit-logs.schema";

const DEFAULT_PAGE_SIZE = 50;
const PREFETCH_PAGES_AHEAD = 2;

export function useAuditLogsQuery(pageSize = DEFAULT_PAGE_SIZE) {
  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));

  const filtersKey = useMemo(
    () => filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|"),
    [filters],
  );

  const prevFiltersKeyRef = useRef(filtersKey);
  useEffect(() => {
    if (prevFiltersKeyRef.current !== filtersKey) {
      prevFiltersKeyRef.current = filtersKey;
      setPage(1);
    }
  }, [filtersKey, setPage]);

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

  const { data, isLoading, isFetching } = trpc.audit.logs.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));

  useEffect(() => {
    if (data && page > totalPages) {
      setPage(totalPages);
    }
  }, [data, page, totalPages, setPage]);

  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = page + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.audit.logs.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [page, totalPages, queryParams, utils.audit.logs]);

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
    auditLogs: data?.auditLogs ?? [],
    isLoading,
    isFetching,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
  };
}
