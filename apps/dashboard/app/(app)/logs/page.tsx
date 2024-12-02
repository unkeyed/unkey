"use server";

import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import {
  createSearchParamsCache,
  parseAsNumberLiteral,
  parseAsString,
  parseAsTimestamp,
} from "nuqs/server";
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

  const logs = await clickhouse.api.logs({
    workspaceId: workspace.id,
    limit: 10,
  });

  //   const logs = await getLogs({
  //     workspaceId: workspace.id,
  //     limit: 100,
  //     startTime,
  //     endTime,
  //     host: parsedParams.host,
  //     requestId: parsedParams.requestId,
  //     method: parsedParams.method,
  //     path: parsedParams.path,
  //     // When responseStatus is missing use "0" to fetch all statuses.
  //     response_status: parsedParams.responseStatus ?? FETCH_ALL_STATUSES,
  //   });

  return <LogsPage initialLogs={logs} workspaceId={workspace.id} />;
}
