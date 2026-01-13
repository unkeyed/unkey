import { parseAsInteger, parseAsString, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { IdentityDetailsFilterValue } from "../filters.schema";

// Define parsers for each filter type
const queryParamsPayload = {
  tags: parseAsString,
  outcomes: parseAsString,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsString,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: IdentityDetailsFilterValue[] = [];

    if (searchParams.tags) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "tags",
        operator: "contains", // Default operator
        value: searchParams.tags,
      });
    }

    if (searchParams.outcomes && KEY_VERIFICATION_OUTCOMES.includes(searchParams.outcomes as any)) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "outcomes",
        operator: "is",
        value: searchParams.outcomes as (typeof KEY_VERIFICATION_OUTCOMES)[number],
      });
    }

    if (searchParams.startTime !== null) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "startTime",
        operator: "is",
        value: searchParams.startTime,
      });
    }

    if (searchParams.endTime !== null) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "endTime",
        operator: "is",
        value: searchParams.endTime,
      });
    }

    if (searchParams.since) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "since",
        operator: "is",
        value: searchParams.since,
      });
    }

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: IdentityDetailsFilterValue[]) => {
      const newParams = {
        tags: null as string | null,
        outcomes: null as string | null,
        startTime: null as number | null,
        endTime: null as number | null,
        since: null as string | null,
      };

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "tags":
            newParams.tags = filter.value;
            break;
          case "outcomes":
            newParams.outcomes = filter.value;
            break;
          case "startTime":
            newParams.startTime = filter.value;
            break;
          case "endTime":
            newParams.endTime = filter.value;
            break;
          case "since":
            newParams.since = filter.value;
            break;
        }
      });

      setSearchParams(newParams);
    },
    [setSearchParams],
  );

  const removeFilter = useCallback(
    (filterId: string) => {
      const updatedFilters = filters.filter((f) => f.id !== filterId);
      updateFilters(updatedFilters);
    },
    [filters, updateFilters],
  );

  const clearFilters = useCallback(() => {
    updateFilters([]);
  }, [updateFilters]);

  return {
    filters,
    updateFilters,
    removeFilter,
    clearFilters,
  };
};