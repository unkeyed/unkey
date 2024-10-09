"use server";

import { getTenantId } from "@/lib/auth";
import { getLogs } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import {
  createSearchParamsCache,
  parseAsArrayOf,
  parseAsNumberLiteral,
  parseAsString,
  parseAsTimestamp,
} from "nuqs/server";
import { generateMockLogs } from "./data";
import LogsPage from "./logs-page";
import { RESPONSE_STATUS_SEPARATOR, STATUSES } from "./query-state";
const mockLogs = generateMockLogs(50);

const searchParamsCache = createSearchParamsCache({
  requestId: parseAsString,
  host: parseAsString,
  method: parseAsString,
  path: parseAsString,
  responseStatutes: parseAsArrayOf(parseAsNumberLiteral(STATUSES), RESPONSE_STATUS_SEPARATOR),
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
  console.log(parsedParams);

  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  const logs = await getLogs({ workspaceId: workspace.id, limit: 10 });
  return <LogsPage logs={logs} />;
}
