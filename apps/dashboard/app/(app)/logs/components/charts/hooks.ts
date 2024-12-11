import { trpc } from "@/lib/trpc/client";
import { LogsTimeseriesDataPoint } from "@unkey/clickhouse/src/logs";
import { useState, useMemo } from "react";
import { useInterval } from "usehooks-ts";
import { useLogSearchParams } from "../../query-state";
import { getTimeseriesGranularity, TimeseriesGranularity } from "../../utils";
import { addMinutes, format } from "date-fns";

const roundToSecond = (timestamp: number) =>
  Math.floor(timestamp / 1000) * 1000;

const formatTimestamp = (
  value: string | number,
  granularity: TimeseriesGranularity
) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);

  switch (granularity) {
    case "perMinute":
      return format(localDate, "HH:mm:ss");
    case "perHour":
      return format(localDate, "MMM d, HH:mm");
    case "perDay":
      return format(localDate, "MMM d");
    default:
      return format(localDate, "Pp");
  }
};

export const useFetchTimeseries = (
  initialTimeseries: LogsTimeseriesDataPoint[]
) => {
  const { searchParams } = useLogSearchParams();

  const filters = useMemo(
    () => ({
      host: searchParams.host,
      path: searchParams.path,
      method: searchParams.method,
      responseStatus: searchParams.responseStatus,
    }),
    [
      searchParams.host,
      searchParams.path,
      searchParams.method,
      searchParams.responseStatus,
    ]
  );

  const {
    startTime: rawStartTime,
    endTime: rawEndTime,
    granularity,
  } = getTimeseriesGranularity(searchParams.startTime, searchParams.endTime);

  const { data, isLoading } = trpc.logs.queryTimeseries.useQuery(
    {
      startTime: roundToSecond(rawStartTime),
      endTime: roundToSecond(rawEndTime),
      ...filters,
    },
    {
      refetchInterval: searchParams.endTime ? false : 10_000,
      initialData: initialTimeseries,
    }
  );

  const timeseries = data.map((data) => ({
    displayX: formatTimestamp(data.x, granularity),
    originalTimestamp: data.x,
    ...data.y,
  }));

  return { timeseries, isLoading };
};
