"use server";

import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { getLogs } from "@unkey/clickhouse/src/logs";
import { createSearchParamsCache } from "nuqs/server";
import { DEFAULT_LOGS_FETCH_COUNT } from "./constants";
import { LogsPage } from "./logs-page";
import { queryParamsPayload } from "./query-state";

const searchParamsCache = createSearchParamsCache(queryParamsPayload);

export default async function Page({
  searchParams,
}: {
  params: { slug: string };
  searchParams: Record<string, string | string[] | undefined>;
}) {
  const parsedParams = searchParamsCache.parse(searchParams);
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  const logs = await getLogs(
    {
      workspaceId: workspace.id,
      limit: DEFAULT_LOGS_FETCH_COUNT,
      startTime: parsedParams.startTime,
      endTime: parsedParams.endTime,
      host: parsedParams.host,
      requestId: parsedParams.requestId,
      method: parsedParams.method,
      path: parsedParams.path,
      responseStatus: parsedParams.responseStatus,
    },
    clickhouse.querier,
  );

  if (logs.err) {
    throw new Error("Something went wrong when fetching logs from clickhouse", logs.err);
  }

  return <LogsPage initialLogs={logs.val} workspaceId={workspace.id} />;
}
