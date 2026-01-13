import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { mapSchemaGranularity } from "@/components/logs/utils";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useMemo } from "react";
import { identityDetailsFilterFieldConfig, type IdentityDetailsFilterValue } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import type { IdentityQueryTimeseriesPayload } from "../query-timeseries.schema";

export const useFetchIdentityTimeseries = (identityId: string) => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: IdentityQueryTimeseriesPayload = {
      identityId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      granularity: "hour", // Default granularity
      outcomes: null,
      tags: null,
      since: "",
    };

    filters.forEach((filter: IdentityDetailsFilterValue) => {
      if (!(filter.field in identityDetailsFilterFieldConfig)) {
        return;
      }

      switch (filter.field) {
        case "tags": {
          if (filter.value.trim()) {
            const fieldConfig = identityDetailsFilterFieldConfig[filter.field];
            const validOperators = fieldConfig.operators;

            const operator = validOperators.includes(filter.operator)
              ? filter.operator
              : validOperators[0];

            params.tags = [
              {
                operator,
                value: filter.value,
              },
            ];
          }
          break;
        }

        case "startTime":
        case "endTime": {
          params[filter.field] = filter.value;
          break;
        }

        case "since": {
          params.since = filter.value;
          break;
        }

        case "outcomes": {
          if (!params.outcomes) {
            params.outcomes = [];
          }
          params.outcomes.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }
      }
    });

    return params;
  }, [filters, timestamp, identityId]);

  const { data, isLoading, isError } = trpc.identity.logs.timeseries.useQuery(queryParams, {
    refetchInterval: queryParams.endTime === timestamp ? 10_000 : false,
    trpc: {
      context: {
        skipBatch: true,
      },
    },
  });

  const timeseries = useMemo(() => {
    if (!data?.timeseries) {
      return [];
    }

    // Convert schema granularity to TimeseriesGranularity format
    const mappedGranularity = mapSchemaGranularity(data.granularity);

    return data.timeseries.map((ts) => {
      const result = {
        displayX: formatTimestampForChart(ts.x, mappedGranularity),
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