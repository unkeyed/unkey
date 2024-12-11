"use server";

import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { createSearchParamsCache } from "nuqs/server";
import { DEFAULT_LOGS_FETCH_COUNT } from "./constants";
import { LogsPage } from "./logs-page";
import { type QuerySearchParams, queryParamsPayload } from "./query-state";
import { getTimeseriesGranularity } from "./utils";
import { notFound } from "next/navigation";

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

  if (!workspace?.betaFeatures.logsPage) {
    return notFound();
  }

  const [logs, timeseries] = await fetchInitialLogsAndTimeseriesData(
    parsedParams,
    workspace.id
  );

  if (timeseries.err) {
    console.error(
      "Error occured when fetching from clickhouse for chart",
      timeseries.err.toString()
    );
    throw new Error(
      "Something went wrong when fetching timeseries data for chart"
    );
  }

  if (logs.err) {
    console.error(
      "Error occured when fetching from clickhouse for table",
      logs.err.toString()
    );
    throw new Error("Something went wrong when fetching logs for table");
  }

  return <LogsPage initialLogs={logs.val} initialTimeseries={timeseries.val} />;
}

const fetchInitialLogsAndTimeseriesData = async (
  params: Readonly<QuerySearchParams>,
  workspaceId: string
) => {
  const { startTime, endTime, granularity } = getTimeseriesGranularity(
    params.startTime,
    params.endTime
  );

  const logs = clickhouse.api.logs({
    workspaceId,
    limit: DEFAULT_LOGS_FETCH_COUNT,
    startTime,
    endTime,
    host: params.host,
    requestId: params.requestId,
    method: params.method,
    path: params.path,
    responseStatus: params.responseStatus,
  });

  const timeseries = clickhouse.api.timeseries[granularity]({
    workspaceId,
    startTime,
    endTime,
    host: params.host,
    method: params.method,
    path: params.path,
    responseStatus: params.responseStatus,
  });

  return Promise.all([logs, timeseries]);
};
