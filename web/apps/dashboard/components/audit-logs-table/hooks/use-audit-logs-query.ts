import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { describeTimeWindow, timeValuesOf } from "@/components/logs/datetime/time-filters";
import { useLoadMoreList } from "@/hooks/use-load-more-list";
import { getTimestampFromRelative } from "@/lib/duration";
import { trpcClient } from "@/lib/trpc/client";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import {
  AUDIT_LOG_BUCKET_UNKEY_MUTATIONS,
  isDashboardAuditLogBucket,
} from "@unkey/schema/src/auditlog";
import { useMemo, useState } from "react";
import type { AuditLogsQueryPayload } from "../schema/audit-logs.schema";

const DEFAULT_PAGE_SIZE = 50;
const DEFAULT_SINCE = "12h";

type AuditLogsPage = inferRouterOutputs<Router>["audit"]["logs"];

const rowsOf = (response: AuditLogsPage) => response.auditLogs;

export function useAuditLogsQuery(pageSize = DEFAULT_PAGE_SIZE) {
  const { filters } = useFilters();

  // The shared query time is set when the dashboard loads, which can predate
  // this page on client-side navigation.
  const { queryTime } = useQueryTime();
  const [mountedAt] = useState(Date.now);
  const anchor = Math.max(queryTime, mountedAt);

  const queryParams = useMemo(() => {
    let since: string | null = null;
    const params: Omit<AuditLogsQueryPayload, "page"> = {
      limit: pageSize,
      startTime: undefined,
      endTime: undefined,
      events: { filters: [] },
      users: { filters: [] },
      rootKeys: { filters: [] },
      since: "",
      bucket: AUDIT_LOG_BUCKET_UNKEY_MUTATIONS,
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
            since = filter.value;
          }
          break;
        }
        case "bucket": {
          if (typeof filter.value === "string" && isDashboardAuditLogBucket(filter.value)) {
            params.bucket = filter.value;
          }
          break;
        }
      }
    }

    if (since !== null) {
      params.startTime = getTimestampFromRelative(since, anchor);
      params.endTime = anchor;
    }
    params.startTime ??= getTimestampFromRelative(DEFAULT_SINCE, anchor);
    params.endTime ??= anchor;

    return params;
  }, [filters, pageSize, anchor]);

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.audit.logs, queryParams, "infinite"),
    fetchPage: (page, signal) => trpcClient.audit.logs.query({ ...queryParams, page }, { signal }),
    rowsOf,
    pageSize,
  });

  return {
    table: list.table,
    totalCount: list.totalCount,
    timeWindowLabel: describeTimeWindow(timeValuesOf(filters), DEFAULT_SINCE),
  };
}
