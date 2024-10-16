"use client";

import { trpc } from "@/lib/trpc/client";
import { LogsChart } from "./components/chart";
import { LogsTable } from "./components/logs-table";
import type { Log } from "./types";
import { useEffect, useState } from "react";
import { LogsFilters } from "./components/filters";
import { FETCH_ALL_STATUSES, ONE_DAY_MS } from "./constants";
import { useLogSearchParams } from "./query-state";

type Props = {
  initialLogs: Log[];
  workspaceId: string;
};

export function LogsPage({ initialLogs, workspaceId }: Props) {
  const { searchParams } = useLogSearchParams();
  const [endTime, setEndTime] = useState<number>(() =>
    searchParams.endTime ? searchParams.endTime.getTime() : Date.now()
  );

  useEffect(() => {
    if (searchParams.endTime) {
      setEndTime(searchParams.endTime.getTime());
    } else {
      const timer = setInterval(() => {
        setEndTime(Date.now());
      }, 3000);
      return () => clearInterval(timer);
    }
  }, [searchParams.endTime]);

  const startTime = searchParams.startTime
    ? searchParams.startTime.getTime()
    : endTime - ONE_DAY_MS;

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
      response_status: searchParams.responseStatus ?? FETCH_ALL_STATUSES,
    },
    {
      refetchInterval: searchParams.endTime ? false : 3000,
      keepPreviousData: true,
    }
  );

  return (
    <div className="flex flex-col gap-4 items-start w-full overflow-y-hidden">
      <LogsFilters />
      <LogsChart logs={logs.data || initialLogs} />
      <LogsTable logs={logs.data || initialLogs} />
    </div>
  );
}
