import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { keyDetailsFilterFieldConfig } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import type { KeyDetailsQueryTimeseriesPayload } from "../query-timeseries.schema";

import { useQuery } from "@tanstack/react-query";

export const useFetchVerificationTimeseries = (keyId: string, keyspaceId: string) => {
  const trpc = useTRPC();
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: KeyDetailsQueryTimeseriesPayload = {
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      outcomes: { filters: [] },
      tags: null,
      since: "",
      keyId,
      keyspaceId,
    };

    filters.forEach((filter) => {
      if (!(filter.field in keyDetailsFilterFieldConfig)) {
        return;
      }

      switch (filter.field) {
        case "tags": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            const fieldConfig = keyDetailsFilterFieldConfig[filter.field];
            const validOperators = fieldConfig.operators;

            const operator = validOperators.includes(filter.operator)
              ? filter.operator
              : validOperators[0];

            params.tags = {
              operator,
              value: filter.value,
            };
          }
          break;
        }

        case "startTime":
        case "endTime": {
          const numValue =
            typeof filter.value === "number"
              ? filter.value
              : typeof filter.value === "string"
                ? Number(filter.value)
                : Number.NaN;

          if (!Number.isNaN(numValue)) {
            params[filter.field] = numValue;
          }
          break;
        }

        case "since": {
          if (typeof filter.value === "string") {
            params.since = filter.value;
          }
          break;
        }

        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes?.filters?.push({
              operator: "is",
              value: filter.value as ValidOutcome,
            });
          }
          break;
        }
      }
    });

    return params;
  }, [filters, timestamp, keyId, keyspaceId]);

  const { data, isLoading, isError } = useQuery(trpc.key.logs.timeseries.queryOptions(queryParams, {
    refetchInterval: queryParams.endTime === timestamp ? 10_000 : false,
    trpc: {
      context: {
        skipBatch: true,
      },
    },
  }));

  const timeseries = useMemo(() => {
    if (!data?.timeseries) {
      return [];
    }

    return data.timeseries.map((ts) => {
      const result = {
        displayX: formatTimestampForChart(ts.x, data.granularity),
        originalTimestamp: ts.x,
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
        outcomeFields.insufficient_permissions = ts.y.insufficient_permissions_count;
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
    granularity: data?.granularity,
  };
};
