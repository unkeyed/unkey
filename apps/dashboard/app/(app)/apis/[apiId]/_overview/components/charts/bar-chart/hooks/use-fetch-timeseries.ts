import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { TIMESERIES_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { keysOverviewFilterFieldConfig } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";
import type { KeysOverviewQueryTimeseriesPayload } from "../query-timeseries.schema";

export const useFetchVerificationTimeseries = (apiId: string | null) => {
  const { filters } = useFilters();
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = useMemo(() => {
    const params: KeysOverviewQueryTimeseriesPayload = {
      startTime: dateNow - TIMESERIES_DATA_WINDOW * 24,
      endTime: dateNow,
      keyIds: { filters: [] },
      outcomes: { filters: [] },
      names: { filters: [] },
      identities: { filters: [] },
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
              operator: "is", // outcomes only support 'is' operator
              value: filter.value as ValidOutcome,
            });
          }
          break;
        }
      }
    });

    return params;
  }, [filters, dateNow, apiId]);

  const { data, isLoading, isError } = trpc.api.keys.timeseries.useQuery(queryParams, {
    refetchInterval: queryParams.endTime === dateNow ? 10_000 : false,
    enabled: Boolean(apiId),
  });

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
