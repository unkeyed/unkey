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
  const [allLogs, setAllLogs] = useState<Log[]>(initialLogs);
  const [latestTimestamp, setLatestTimestamp] = useState<number | null>(null);
  const [endTime, setEndTime] = useState<number>(() =>
    searchParams.endTime ? searchParams.endTime.getTime() : Date.now()
  );

  useEffect(() => {
    if (searchParams.endTime) {
      setEndTime(searchParams.endTime.getTime());
      setLatestTimestamp(null);
      setAllLogs(initialLogs);
    } else {
      const timer = setInterval(() => {
        setEndTime(Date.now());
      }, 3000);
      return () => clearInterval(timer);
    }
  }, [searchParams.endTime, initialLogs]);

  const startTime =
    latestTimestamp ||
    (searchParams.startTime
      ? searchParams.startTime.getTime()
      : endTime - ONE_DAY_MS);

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

  useEffect(() => {
    if (logs.data?.length) {
      const newLatestTimestamp = Math.max(
        ...logs.data.map((log) => new Date(log.time).getTime())
      );
      setLatestTimestamp(newLatestTimestamp);

      setAllLogs((prevLogs) => {
        const newLogs = logs.data.filter(
          (log) =>
            !prevLogs.some((prevLog) => prevLog.request_id === log.request_id)
        );
        return [...newLogs, ...prevLogs];
      });
    }
  }, [logs.data]);

  return (
    <div className="flex flex-col gap-4 items-start w-full overflow-y-hidden">
      <LogsFilters />
      <LogsChart logs={allLogs} />
      <LogsTable logs={allLogs} />
    </div>
  );
}
