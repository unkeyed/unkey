import type { LogsRequestSchema } from "@/lib/schemas/logs.schema";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useState } from "react";

const REQUEST_DETAILS_TIME_BUFFER_MS = 60_000;
const MISSING_LOG_RETRY_INTERVAL_MS = 2_000;
const MISSING_LOG_MAX_ATTEMPTS = 6;

type RequestDetailsTarget = {
  requestId?: string;
  /** Time of the row the request belongs to; anchors the lookup window. */
  time?: number;
};

export function buildRequestDetailsQueryParams({
  requestId,
  time,
}: RequestDetailsTarget): LogsRequestSchema {
  const anchor = time ?? 0;

  return {
    limit: 1,
    startTime: Math.max(0, anchor - REQUEST_DETAILS_TIME_BUFFER_MS),
    endTime: anchor + REQUEST_DETAILS_TIME_BUFFER_MS,
    host: { filters: [] },
    method: { filters: [] },
    path: { filters: [] },
    status: { filters: [] },
    requestId: requestId
      ? {
          filters: [
            {
              operator: "is",
              value: requestId,
            },
          ],
        }
      : null,
    since: "",
  };
}

export function useFetchRequestDetails({ requestId, time }: RequestDetailsTarget) {
  const [missCount, setMissCount] = useState(0);

  const enabled = Boolean(requestId) && time !== undefined;
  const retryScheduled = enabled && missCount > 0 && missCount < MISSING_LOG_MAX_ATTEMPTS;

  const query = trpc.logs.queryLogs.useQuery(buildRequestDetailsQueryParams({ requestId, time }), {
    enabled,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    staleTime: Number.POSITIVE_INFINITY,
    refetchInterval: retryScheduled ? MISSING_LOG_RETRY_INTERVAL_MS : false,
  });

  // missCount only covers responses the effect below has already folded in, so
  // it lags the response being rendered now. Judging the current response keeps
  // the first empty result from reading as settled before a retry has run.
  const isAwaitingIngestion =
    enabled && query.isSuccess && !query.data?.logs.length && missCount < MISSING_LOG_MAX_ATTEMPTS;

  const [trackedRequestId, setTrackedRequestId] = useState(requestId);
  if (trackedRequestId !== requestId) {
    setTrackedRequestId(requestId);
    setMissCount(0);
  }

  const { data, dataUpdatedAt } = query;
  useEffect(() => {
    if (!dataUpdatedAt) {
      return;
    }
    setMissCount((count) => (data?.logs.length ? 0 : count + 1));
  }, [data, dataUpdatedAt]);

  return {
    log: query.data?.logs[0],
    isLoading: query.isLoading || isAwaitingIngestion,
    error: query.error,
  };
}
