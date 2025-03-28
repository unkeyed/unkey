import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

export const useFetchVerificationTimeseries = (
  keyAuthId: string,
  keyId: string
) => {
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = {
    startTime: dateNow - HISTORICAL_DATA_WINDOW * 3,
    endTime: dateNow,
    keyAuthId,
    keyId,
  };

  const { data, isLoading, isError } = trpc.api.keys.usageTimeseries.useQuery(
    queryParams,
    {
      refetchInterval: queryParams.endTime === dateNow ? 10_000 : false,
    }
  );

  const timeseries = useMemo(() => {
    if (!data?.timeseries) {
      return [];
    }
    return data.timeseries.map((ts) => {
      const result = {
        valid: ts.y.valid,
        total: ts.y.total,
        success: ts.y.valid,
        error: ts.y.total - ts.y.valid,
      };
      const outcomeFields: Record<string, number> = {};
      if (ts.y.rate_limited_count !== undefined) {
        outcomeFields.rate_limited = ts.y.rate_limited_count;
      }
      if (ts.y.insufficient_permissions_count !== undefined) {
        outcomeFields.insufficient_permissions =
          ts.y.insufficient_permissions_count;
      }
      if (ts.y.forbidden_count !== undefined) {
        outcomeFields.forbidden = ts.y.forbidden_count;
      }
      if (ts.y.disabled_count !== undefined) {
        outcomeFields.disabled = ts.y.disabled_count;
      }
      if (ts.y.expired_count !== undefined) {
        outcomeFields.expired = ts.y.expired_count;
      }
      if (ts.y.usage_exceeded_count !== undefined) {
        outcomeFields.usage_exceeded = ts.y.usage_exceeded_count;
      }
      return {
        ...result,
        ...outcomeFields,
      };
    });
  }, [data]);

  return {
    timeseries: timeseries || [],
    isLoading,
    isError,
  };
};
