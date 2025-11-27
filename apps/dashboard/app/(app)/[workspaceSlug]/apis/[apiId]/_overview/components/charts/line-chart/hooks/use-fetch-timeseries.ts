import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { keysOverviewFilterFieldConfig } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import type { KeysOverviewQueryTimeseriesPayload } from "../../bar-chart/query-timeseries.schema";

import { useQuery } from "@tanstack/react-query";

export const useFetchActiveKeysTimeseries = (apiId: string | null) => {
  const trpc = useTRPC();
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: KeysOverviewQueryTimeseriesPayload = {
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      keyIds: { filters: [] },
      outcomes: { filters: [] },
      names: { filters: [] },
      identities: { filters: [] },
      tags: null,
      apiId: apiId ?? "",
      since: "",
    };

    if (!apiId) {
      return params;
    }

    filters.forEach((filter) => {
      if (!(filter.field in keysOverviewFilterFieldConfig)) {
        return;
      }

      const fieldConfig = keysOverviewFilterFieldConfig[filter.field];
      const validOperators = fieldConfig.operators;

      const operator = validOperators.includes(filter.operator)
        ? filter.operator
        : validOperators[0];

      switch (filter.field) {
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

        case "keyIds": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            const keyIdOperator = operator === "is" || operator === "contains" ? operator : "is";

            params.keyIds?.filters?.push({
              operator: keyIdOperator,
              value: filter.value,
            });
          }
          break;
        }

        case "names":
        case "identities": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            params[filter.field]?.filters?.push({
              operator,
              value: filter.value,
            });
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

        case "tags": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            params.tags = {
              operator,
              value: filter.value,
            };
          }
          break;
        }
      }
    });

    return params;
  }, [filters, timestamp, apiId]);

  const { data, isLoading, isError } = useQuery(trpc.api.keys.activeKeysTimeseries.queryOptions(queryParams, {
    refetchInterval: queryParams.endTime === timestamp ? 10_000 : false,
    enabled: Boolean(apiId),
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
      return {
        displayX: formatTimestampForChart(ts.x, data.granularity),
        originalTimestamp: ts.x,
        keys: ts.y.keys,
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
