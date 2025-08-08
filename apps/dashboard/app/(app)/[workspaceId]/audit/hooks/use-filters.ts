import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, parseAsString, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import type {
  AuditLogsFilterField,
  AuditLogsFilterOperator,
  AuditLogsFilterUrlValue,
  AuditLogsFilterValue,
  QuerySearchParams,
} from "../filters.schema";

const parseAsFilterValArray = parseAsFilterValueArray<AuditLogsFilterOperator>(["is"]);

export const auditQueryParamsPayload = {
  events: parseAsFilterValArray,
  users: parseAsFilterValArray,
  rootKeys: parseAsFilterValArray,
  bucket: parseAsString,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(auditQueryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: AuditLogsFilterValue[] = [];

    searchParams.events?.forEach((event) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "events",
        operator: event.operator,
        value: event.value,
      });
    });

    searchParams.users?.forEach((user) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "users",
        operator: user.operator,
        value: user.value,
      });
    });

    searchParams.rootKeys?.forEach((rootKey) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "rootKeys",
        operator: rootKey.operator,
        value: rootKey.value,
      });
    });

    // Handle bucket as string directly
    if (searchParams.bucket) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "bucket",
        operator: "is",
        value: searchParams.bucket,
      });
    }

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof QuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as AuditLogsFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: AuditLogsFilterValue[]) => {
      const newParams: Partial<QuerySearchParams> = {
        events: null,
        users: null,
        rootKeys: null,
        bucket: null,
        startTime: null,
        endTime: null,
        since: null,
      };

      // Group filters by field
      const eventsFilters: AuditLogsFilterUrlValue[] = [];
      const usersFilters: AuditLogsFilterUrlValue[] = [];
      const rootKeysFilters: AuditLogsFilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "events":
            eventsFilters.push({
              value: filter.value as string,
              operator: filter.operator,
            });
            break;
          case "users":
            usersFilters.push({
              value: filter.value as string,
              operator: filter.operator,
            });
            break;
          case "rootKeys":
            rootKeysFilters.push({
              value: filter.value as string,
              operator: filter.operator,
            });
            break;
          case "bucket":
            newParams.bucket = filter.value as string;
            break;
          case "startTime":
          case "endTime":
            newParams[filter.field] = filter.value as number;
            break;
          case "since":
            newParams.since = filter.value as string;
            break;
        }
      });

      // Set arrays to null when empty, otherwise use the filtered values
      newParams.events = eventsFilters.length > 0 ? eventsFilters : null;
      newParams.users = usersFilters.length > 0 ? usersFilters : null;
      newParams.rootKeys = rootKeysFilters.length > 0 ? rootKeysFilters : null;

      setSearchParams(newParams);
    },
    [setSearchParams],
  );

  const removeFilter = useCallback(
    (id: string) => {
      const newFilters = filters.filter((f) => f.id !== id);
      updateFilters(newFilters);
    },
    [filters, updateFilters],
  );

  return {
    filters,
    removeFilter,
    updateFilters,
  };
};
