import { trpc } from "@/lib/trpc/client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useDebounceCallback, useInterval } from "usehooks-ts";
import { useLogSearchParams } from "../../query-state";
import { Log } from "../../types";
import { getTimeseriesGranularity } from "../../utils";

const roundToSecond = (timestamp: number) =>
  Math.floor(timestamp / 1000) * 1000;

export const useFetchLogs = (initialLogs: Log[]) => {
  const { searchParams, setSearchParams } = useLogSearchParams();
  const [logs, setLogs] = useState(initialLogs);
  const [endTime, setEndTime] = useState(searchParams.endTime);

  useInterval(() => setEndTime(Date.now()), searchParams.endTime ? null : 5000);

  const filters = useMemo(
    () => ({
      host: searchParams.host,
      requestId: searchParams.requestId,
      path: searchParams.path,
      method: searchParams.method,
      responseStatus: searchParams.responseStatus,
    }),
    [
      searchParams.host,
      searchParams.requestId,
      searchParams.path,
      searchParams.method,
      searchParams.responseStatus,
    ]
  );

  const hasFilters = useMemo(
    () =>
      Boolean(
        filters.host ||
          filters.requestId ||
          filters.path ||
          filters.method ||
          filters.responseStatus.length
      ),
    [filters]
  );

  const { startTime: rawStartTime, endTime: rawEndTime } =
    getTimeseriesGranularity(searchParams.startTime, endTime);

  const { data: newData, isLoading } = trpc.logs.queryLogs.useQuery(
    {
      limit: 100,
      startTime: roundToSecond(rawStartTime),
      endTime: roundToSecond(rawEndTime),
      ...filters,
    },
    {
      refetchInterval: searchParams.endTime ? false : 5000,
      keepPreviousData: true,
    }
  );

  const updateLogs = useCallback(() => {
    if (hasFilters) {
      setLogs(newData ?? []);
      return;
    }

    if (!newData?.length) {
      return;
    }

    setLogs((prevLogs) => {
      const existingIds = new Set(prevLogs.map((log) => log.request_id));
      const uniqueNewLogs = newData.filter(
        (newLog) => !existingIds.has(newLog.request_id)
      );
      return [...prevLogs, ...uniqueNewLogs];
    });
  }, [newData, hasFilters]);

  const handleQueryParamReset = () => {
    setSearchParams({
      host: null,
      path: null,
      method: null,
      endTime: null,
      requestId: null,
      startTime: null,
      responseStatus: null,
    });
  };

  const switchToLive = useDebounceCallback(handleQueryParamReset, 350);

  useEffect(() => {
    updateLogs();
  }, [updateLogs]);

  return { logs, isLoading, switchToLive };
};
