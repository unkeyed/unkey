import type { LogsRequestSchema } from "@/lib/schemas/logs.schema";
import { trpc } from "@/lib/trpc/client";
import type { RequestDetailsRequest, RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import type { Log } from "@unkey/clickhouse/src/logs";
import type { KeyVerificationSource } from "@unkey/clickhouse/src/verifications";
import { useEffect, useState } from "react";

const REQUEST_DETAILS_TIME_BUFFER_MS = 60_000;
const MISSING_LOG_RETRY_INTERVAL_MS = 2_000;
const MISSING_LOG_MAX_ATTEMPTS = 6;

type RequestDetailsTarget = {
  requestId?: string;
  /** Time of the row the request belongs to; anchors the lookup window. */
  time?: number;
  /**
   * Which service recorded the verification. `api` requests are logged to
   * api_requests_raw_v2, `gateway` ones to frontline_requests_raw_v1, so the
   * source decides which table holds the matching request.
   */
  source?: KeyVerificationSource;
};

/** The request row, tagged with the table it came from so callers can render it. */
export type RequestDetails =
  | { source: "api"; log: Log }
  | { source: "gateway"; log: RequestLogsResponse };

function requestDetailsWindow(time: number | undefined) {
  const anchor = time ?? 0;

  return {
    startTime: Math.max(0, anchor - REQUEST_DETAILS_TIME_BUFFER_MS),
    endTime: anchor + REQUEST_DETAILS_TIME_BUFFER_MS,
  };
}

export function buildRequestDetailsQueryParams({
  requestId,
  time,
}: RequestDetailsTarget): LogsRequestSchema {
  return {
    limit: 1,
    ...requestDetailsWindow(time),
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

export function buildGatewayRequestDetailsQueryParams({
  requestId,
  time,
}: RequestDetailsTarget): Omit<RequestDetailsRequest, "workspaceId"> {
  return {
    // The query only runs when a request is selected; the placeholder keeps the
    // input schema satisfied for the disabled render.
    requestId: requestId ?? "",
    ...requestDetailsWindow(time),
  };
}

export function useFetchRequestDetails({ requestId, time, source }: RequestDetailsTarget) {
  const [missCount, setMissCount] = useState(0);

  const isGateway = source === "gateway";
  const enabled = Boolean(requestId) && time !== undefined;
  const retryScheduled = enabled && missCount > 0 && missCount < MISSING_LOG_MAX_ATTEMPTS;

  const sharedOptions = {
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    staleTime: Number.POSITIVE_INFINITY,
    // The attempt budget below is the retry: react-query's own attempts would
    // stack extra requests inside an interval tick without advancing it.
    retry: false,
    refetchInterval: retryScheduled ? MISSING_LOG_RETRY_INTERVAL_MS : (false as const),
  };

  const apiQuery = trpc.logs.queryLogs.useQuery(
    buildRequestDetailsQueryParams({ requestId, time }),
    {
      ...sharedOptions,
      enabled: enabled && !isGateway,
    },
  );

  const gatewayQuery = trpc.deploy.requestLogs.details.useQuery(
    buildGatewayRequestDetailsQueryParams({ requestId, time }),
    {
      ...sharedOptions,
      enabled: enabled && isGateway,
    },
  );

  const query = isGateway ? gatewayQuery : apiQuery;
  const found: RequestDetails | undefined = isGateway
    ? gatewayQuery.data?.log
      ? { source: "gateway", log: gatewayQuery.data.log }
      : undefined
    : apiQuery.data?.logs[0]
      ? { source: "api", log: apiQuery.data.logs[0] }
      : undefined;

  // missCount only covers responses the effect below has already folded in, so
  // it lags the response being rendered now. Judging the current response keeps
  // the first empty result from reading as settled before a retry has run.
  const isAwaitingIngestion =
    enabled && (query.isSuccess || query.isError) && !found && missCount < MISSING_LOG_MAX_ATTEMPTS;

  const [trackedRequestId, setTrackedRequestId] = useState(requestId);
  if (trackedRequestId !== requestId) {
    setTrackedRequestId(requestId);
    setMissCount(0);
  }

  const { dataUpdatedAt, errorUpdatedAt } = query;
  // A failed poll leaves data and dataUpdatedAt untouched, so errors have to
  // spend the budget too, or refetchInterval would keep firing forever.
  const settledAt = Math.max(dataUpdatedAt, errorUpdatedAt);
  // `found` is rebuilt every render, so the effect keys off the boolean to stay
  // idle until a response actually settles.
  const foundLog = Boolean(found);
  useEffect(() => {
    if (!settledAt) {
      return;
    }
    setMissCount((count) => (foundLog ? 0 : count + 1));
  }, [foundLog, settledAt]);

  return {
    details: found,
    isLoading: query.isLoading || isAwaitingIngestion,
    error: query.error,
  };
}
