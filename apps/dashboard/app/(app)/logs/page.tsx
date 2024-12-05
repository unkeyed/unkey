"use server";

import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { createSearchParamsCache } from "nuqs/server";
import { DEFAULT_LOGS_FETCH_COUNT } from "./constants";
import { LogsPage } from "./logs-page";
import { queryParamsPayload } from "./query-state";
import { hasWorkspaceAccess } from "@/lib/utils";

const searchParamsCache = createSearchParamsCache(queryParamsPayload);

export default async function Page({
  searchParams,
}: {
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

  if (!hasWorkspaceAccess("logsPage", workspace)) {
    return notFound();
  }

  const logs = await clickhouse.api.logs({
    workspaceId: workspace.id,
    limit: DEFAULT_LOGS_FETCH_COUNT,
    startTime: parsedParams.startTime,
    endTime: parsedParams.endTime ?? Date.now(),
    host: parsedParams.host,
    requestId: parsedParams.requestId,
    method: parsedParams.method,
    path: parsedParams.path,
    responseStatus: parsedParams.responseStatus,
  });

  if (logs.err) {
    throw new Error(
      `Something went wrong when fetching logs from ClickHouse: ${logs.err.message}`
    );
  }

  return <LogsPage initialLogs={logs.val} workspaceId={workspace.id} />;
}
