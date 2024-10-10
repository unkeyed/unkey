"use server";

import { getTenantId } from "@/lib/auth";
import { getLogs } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import {
  createSearchParamsCache,
  parseAsNumberLiteral,
  parseAsString,
  parseAsTimestamp,
} from "nuqs/server";
import LogsPage from "./logs-page";
import { STATUSES } from "./query-state";

const ONE_DAY_MS = 24 * 60 * 60 * 1000; // ms in a day
const FETCH_ALL_STATUSES = 0;
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

  const now = Date.now();
  const startTime = parsedParams.startTime ? parsedParams.startTime.getTime() : now - ONE_DAY_MS;
  const endTime = parsedParams.endTime ? parsedParams.endTime.getTime() : Date.now();

  const logs = await getLogs({
    workspaceId: workspace.id,
    limit: 100,
    startTime,
    endTime,
    host: parsedParams.host,
    requestId: parsedParams.requestId,
    method: parsedParams.method,
    path: parsedParams.path,
    // When responseStatus is missing use "0" to fetch all statuses.
    response_status: parsedParams.responseStatus ?? FETCH_ALL_STATUSES,
  });

  return <LogsPage logs={logs} />;
}
