"use server";

import { getTenantId } from "@/lib/auth";
import { getLogs } from "@/lib/clickhouse/logs";
import { db } from "@/lib/db";
import {
  createSearchParamsCache,
  parseAsNumberLiteral,
  parseAsString,
  parseAsTimestamp,
} from "nuqs/server";
import { FETCH_ALL_STATUSES } from "./constants";
import { LogsPage } from "./logs-page";
import { STATUSES } from "./query-state";

const searchParamsCache = createSearchParamsCache({
  requestId: parseAsString,
  host: parseAsString,
  method: parseAsString,
  path: parseAsString,
  responseStatus: parseAsNumberLiteral(STATUSES),
  startTime: parseAsTimestamp,
  endTime: parseAsTimestamp,
});

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

  const logs = await getLogs({
    workspaceId: workspace.id,
    limit: 100,
    startTime: parsedParams.startTime?.getTime() ?? Date.now(),
    endTime: parsedParams.endTime?.getTime() ?? Date.now(),
    host: parsedParams.host,
    requestId: parsedParams.requestId,
    method: parsedParams.method,
    path: parsedParams.path,
    // When responseStatus is missing use "0" to fetch all statuses.
    response_status: parsedParams.responseStatus ?? FETCH_ALL_STATUSES,
  });
  if (logs.err) {
    throw new Error(
      "Something went wrong when fetching logs from clickhouse",
      logs.err
    );
  }

  return <LogsPage initialLogs={logs.val} workspaceId={workspace.id} />;
}
