"use client";

import { trpc } from "@/lib/trpc/client";
import { LogsChart } from "./components/chart";
import { LogsTable } from "./components/logs-table";
import type { Log } from "./data";

import { useLogSearchParams } from "./query-state";
import { LogsFilters } from "./components/filters";

type Props = {
  logs: Log[];
  workspaceId: string;
};

const ONE_DAY_MS = 24 * 60 * 60 * 1000; // ms in a day
const FETCH_ALL_STATUSES = 0;

export function LogsPage({ logs: _logs, workspaceId }: Props) {
  const { searchParams } = useLogSearchParams();

  const now = Date.now();
  const startTime = searchParams.startTime
    ? searchParams.startTime.getTime()
    : now - ONE_DAY_MS;
  const endTime = searchParams.endTime
    ? searchParams.endTime.getTime()
    : Date.now();

  const logs = trpc.logs.queryLogs.useQuery(
    {
      workspaceId,
      limit: 100,
      startTime,
      endTime,
      host: searchParams.host,
      requestId: searchParams.requestId,
      method: searchParams.method,
      path: searchParams.path,
      // When responseStatus is missing use "0" to fetch all statuses.
      response_status: searchParams.responseStatus ?? FETCH_ALL_STATUSES,
    },
    { initialData: _logs }
  );

  return (
    <div className="flex flex-col gap-4 items-start w-full overflow-y-hidden">
      <LogsFilters />
      <LogsChart logs={logs.data} />
      <LogsTable logs={logs.data} isLoading={logs.isLoading} />
    </div>
  );
}
