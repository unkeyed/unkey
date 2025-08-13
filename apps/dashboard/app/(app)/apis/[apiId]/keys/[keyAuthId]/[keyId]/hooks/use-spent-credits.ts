import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { keyDetailsFilterFieldConfig } from "../filters.schema";
import { useFilters } from "./use-filters";

export const useSpentCredits = (keyId: string, keyspaceId: string) => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params = {
      keyId,
      keyspaceId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      outcomes: [] as Array<{
        value:
          | "VALID"
          | "RATE_LIMITED"
          | "INSUFFICIENT_PERMISSIONS"
          | "FORBIDDEN"
          | "DISABLED"
          | "EXPIRED"
          | "USAGE_EXCEEDED";
        operator: "is";
      }>,
      tags: null as {
        operator: "is" | "contains" | "startsWith" | "endsWith";
        value: string;
      } | null,
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

        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            filter.value !== "" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes.push({
              operator: "is" as const,
              value: filter.value as Exclude<ValidOutcome, "">,
            });
          }
          break;
        }
      }
    });

    return {
      ...params,
      outcomes: params.outcomes.length > 0 ? params.outcomes : null,
    };
  }, [filters, timestamp, keyId, keyspaceId]);

  const { data, isLoading, isError } = trpc.key.spentCredits.useQuery(queryParams, {
    refetchInterval: queryParams.endTime === timestamp ? 10_000 : false,
  });

  return {
    spentCredits: data?.spentCredits ?? 0,
    isLoading,
    isError,
  };
};
