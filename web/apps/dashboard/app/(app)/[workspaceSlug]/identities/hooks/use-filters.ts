import { parseAsRelativeTime } from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type IdentitiesFilterValue,
  identitiesFilterFieldConfig,
  identitiesListFilterFieldNames,
  parseAsIdentitiesFilterArray,
} from "../filters.schema";

const queryParamsPayload = {
  externalId: parseAsIdentitiesFilterArray,
  lastUsedStart: parseAsInteger,
  lastUsedEnd: parseAsInteger,
  lastUsedSince: parseAsRelativeTime,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: IdentitiesFilterValue[] = [];

    // Array-type filters (externalId)
    for (const field of identitiesListFilterFieldNames) {
      const value = searchParams[field];
      if (!Array.isArray(value)) {
        continue;
      }
      value.forEach((filterItem, idx) => {
        const operator = filterItem?.operator;
        const raw = typeof filterItem?.value === "string" ? filterItem.value : "";
        const v = raw.trim();
        const allowed = identitiesFilterFieldConfig[field]?.operators ?? [];
        if (!v || !operator || !allowed.includes(operator as never)) {
          return;
        }
        activeFilters.push({
          id: `${field}:${operator}:${v}:${idx}`,
          field,
          operator,
          value: v,
        });
      });
    }

    // Scalar datetime filters
    if (searchParams.lastUsedStart !== null && searchParams.lastUsedStart !== undefined) {
      activeFilters.push({
        id: `lastUsedStart:is:${searchParams.lastUsedStart}:0`,
        field: "lastUsedStart",
        operator: "is",
        value: searchParams.lastUsedStart,
      });
    }
    if (searchParams.lastUsedEnd !== null && searchParams.lastUsedEnd !== undefined) {
      activeFilters.push({
        id: `lastUsedEnd:is:${searchParams.lastUsedEnd}:0`,
        field: "lastUsedEnd",
        operator: "is",
        value: searchParams.lastUsedEnd,
      });
    }
    if (searchParams.lastUsedSince !== null && searchParams.lastUsedSince !== undefined) {
      activeFilters.push({
        id: `lastUsedSince:is:${searchParams.lastUsedSince}:0`,
        field: "lastUsedSince",
        operator: "is",
        value: searchParams.lastUsedSince,
      });
    }

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: IdentitiesFilterValue[]) => {
      const newParams: {
        externalId: AllOperatorsUrlValue[] | null;
        lastUsedStart: number | null;
        lastUsedEnd: number | null;
        lastUsedSince: string | null;
      } = {
        externalId: null,
        lastUsedStart: null,
        lastUsedEnd: null,
        lastUsedSince: null,
      };

      const externalIdFilters: AllOperatorsUrlValue[] = [];

      for (const filter of newFilters) {
        switch (filter.field) {
          case "externalId": {
            if (typeof filter.value === "string") {
              externalIdFilters.push({ value: filter.value, operator: filter.operator });
            }
            break;
          }
          case "lastUsedStart":
          case "lastUsedEnd": {
            newParams[filter.field] = filter.value as number;
            break;
          }
          case "lastUsedSince": {
            newParams.lastUsedSince = filter.value as string;
            break;
          }
        }
      }

      if (externalIdFilters.length > 0) {
        newParams.externalId = externalIdFilters;
      }

      setSearchParams(newParams);
    },
    [setSearchParams],
  );

  const removeFilter = useCallback(
    (id: string) => {
      updateFilters(filters.filter((f) => f.id !== id));
    },
    [filters, updateFilters],
  );

  return { filters, removeFilter, updateFilters };
};
