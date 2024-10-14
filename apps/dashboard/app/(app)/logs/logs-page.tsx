"use client";

import { trpc } from "@/lib/trpc/client";
import { LogsChart } from "./components/chart";
import { LogsTable } from "./components/logs-table";
import type { Log } from "./types";

import { useEffect, useMemo, useState } from "react";
import { LogsFilters } from "./components/filters";
import { FETCH_ALL_STATUSES, ONE_DAY_MS } from "./constants";
import { useLogSearchParams } from "./query-state";

type Props = {
  logs: Log[];
  workspaceId: string;
};

export function LogsPage({ logs: _logs, workspaceId }: Props) {
  const { searchParams } = useLogSearchParams();
  const [endTime, setEndTime] = useState(
    searchParams.endTime ? searchParams.endTime.getTime() : Date.now(),
  );

  const now = useMemo(() => Date.now(), []);
  const startTime = searchParams.startTime ? searchParams.startTime.getTime() : now - ONE_DAY_MS;

  useEffect(() => {
    if (searchParams.endTime) {
      setEndTime(searchParams.endTime.getTime());
      return;
    }
    const timer = setInterval(() => {
      setEndTime(Date.now());
    }, 3000);
    return () => {
      clearInterval(timer);
    };
  }, [searchParams.endTime]);

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
    { keepPreviousData: true, initialData: _logs },
  );

  return (
    <div className="flex flex-col gap-4 items-start w-full overflow-y-hidden">
      <LogsFilters />
      <LogsChart logs={logs.data} />
      <LogsTable logs={logs.data} />
    </div>
  );
}
