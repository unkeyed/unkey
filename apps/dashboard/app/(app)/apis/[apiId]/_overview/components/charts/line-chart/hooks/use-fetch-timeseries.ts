import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { TIMESERIES_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { useFilters } from "../../../../hooks/use-filters";
import type { KeysOverviewQueryTimeseriesPayload } from "../../bar-chart/query-timeseries.schema";

export const useFetchActiveKeysTimeseries = (apiId: string | null) => {
  const { filters } = useFilters();
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = useMemo(() => {
    const params: KeysOverviewQueryTimeseriesPayload = {
      startTime: dateNow - TIMESERIES_DATA_WINDOW * 24,
      endTime: dateNow,
      keyIds: { filters: [] },
      outcomes: { filters: [] },
      names: { filters: [] },
      apiId: apiId ?? "",
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'number'`);
            return;
          }
          params[filter.field] = filter.value;
          break;
        }
        case "since": {
          if (typeof filter.value !== "string") {
            console.error("Since filter value type has to be 'string'");
            return;
          }
          params.since = filter.value;
          break;
        }
        case "keyIds": {
          if (typeof filter.value !== "string") {
            console.error("Keys filter value type has to be 'string'");
            return;
          }
          params.keyIds?.filters?.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
        case "names": {
          if (typeof filter.value !== "string") {
            console.error("Names filter value type has to be 'string'");
            return;
          }
          params.names?.filters?.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
        case "outcomes": {
          //TODO: later use enum from zod type in clickhouse

          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes?.filters?.push({
              operator: "is",
              value: filter.value as ValidOutcome,
            });
          } else {
            console.error("Invalid outcome value:", filter.value);
          }
          break;
        }
      }
    });

    return params;
  }, [filters, dateNow, apiId]);

  const { data, isLoading, isError } = trpc.api.keys.activeKeysTimeseries.useQuery(queryParams, {
    refetchInterval: queryParams.endTime ? false : 10_000,
  });

  // Process timeseries data to work with our chart component
  const timeseries = data?.timeseries.map((ts) => {
    return {
      displayX: formatTimestampForChart(ts.x, data.granularity),
      originalTimestamp: ts.x,
      keys: ts.y.keys,
    };
  });

  return {
    timeseries,
    isLoading,
    isError,
    granularity: data?.granularity,
  };
};
