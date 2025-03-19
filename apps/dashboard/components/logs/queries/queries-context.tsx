import type { QuerySearchParams } from "@/app/(app)/logs/filters.schema";
import { type ReactNode, createContext, useContext } from "react";
import { type SavedFiltersGroup, useBookmarkedFilters } from "../hooks/use-bookmarked-filters";
import type { FilterValue } from "../validation/filter.types";

type QueriesContextType = {
  formatValues: (
    filters: QuerySearchParams,
  ) => Record<string, { operator: string; values: { value: string; color: string | null }[] }>;
  filterGroups: SavedFiltersGroup<QuerySearchParams>[];
  toggleBookmark: (groupId: string) => void;
  applyFilterGroup: (groupId: string) => void;
  filterRowIcon: (field: string) => React.ReactNode;
  shouldTruncateRow: (field: string) => boolean;
};

const QueriesContext = createContext<QueriesContextType | undefined>(undefined);

type QueriesProviderProps<T extends FilterValue> = {
  children: ReactNode;
  localStorageName: string;
  filters: T[];
  updateFilters: (filters: T[]) => void;
  formatValues?: (
    filters: QuerySearchParams,
  ) => Record<string, { operator: string; values: { value: string; color: string | null }[] }>;
  filterRowIcon?: (field: string) => React.ReactNode;
  shouldTruncateRow?: (field: string) => boolean;
};

export function QueriesProvider<T extends FilterValue>({
  children,
  localStorageName,
  filters,
  updateFilters,
  formatValues,
  filterRowIcon,
  shouldTruncateRow,
}: QueriesProviderProps<T>) {
  const {
    savedFilters,
    toggleBookmark,
    applyFilterGroup: originalApplyFilterGroup,
  } = useBookmarkedFilters({
    localStorageName,
    filters,
    updateFilters,
  });

  const filterGroups = savedFilters;

  const applyFilterGroup = (groupId: string) => {
    const group = savedFilters.find((g) => g.id === groupId);
    if (group) {
      originalApplyFilterGroup(group);
    }
  };

  return (
    <QueriesContext.Provider
      value={{
        formatValues: formatValues || defaultFormatValues,
        filterGroups,
        toggleBookmark,
        applyFilterGroup,
        filterRowIcon: filterRowIcon || defaultGetIcon,
        shouldTruncateRow: shouldTruncateRow || defaultShouldTruncateRow,
      }}
    >
      {children}
    </QueriesContext.Provider>
  );
}

export function useQueries() {
  const context = useContext(QueriesContext);
  if (context === undefined) {
    throw new Error("useQueries must be used within a QueriesProvider");
  }
  return context as QueriesContextType;
}

import { ChartActivity2 } from "@unkey/icons";
import React from "react";
import { iconsPerField } from "./utils";
// These functions can be overridden by passing custom formatValue and filterRowIcon props to QueriesProvider
export const defaultFormatValues = (
  filters: QuerySearchParams,
): Record<string, { operator: string; values: { value: string; color: string | null }[] }> => {
  const transform = (field: string, value: string): { color: string | null; value: string } => {
    switch (field) {
      case "status":
        return {
          value:
            value === "200" ? "2xx" : value === "400" ? "4xx" : value === "500" ? "5xx" : value,
          color:
            value === "200"
              ? "bg-success-9"
              : value === "400"
                ? "bg-warning-9"
                : value === "500"
                  ? "bg-error-9"
                  : null,
        };
      case "methods":
        return { value: value.toUpperCase(), color: null };
      case "startTime":
      case "endTime":
      case "since":
        return { value: new Date(Number(value)).toLocaleString(), color: null };
      case "host":
      case "requestId":
      case "paths":
        return { value: value, color: null };
      default:
        return { value: value, color: null };
    }
  };
  const transformed: Record<
    string,
    { operator: string; values: { value: string; color: string | null }[] }
  > = {};
  // Handle special cases for different field types const transformed: Record<string, { operator: string; values: { value: string; color: string | null }[], icon: React.ReactNode }> = {};
  if (filters.startTime && filters.endTime) {
    transformed.time = {
      operator: "between",
      values: [
        transform("startTime", filters.startTime.toString()),
        transform("endTime", filters.endTime.toString()),
      ],
    };
  }
  if (filters.startTime) {
    transformed.startTime = {
      operator: "starts with",
      values: [transform("startTime", filters.startTime.toString())],
    };
  }
  if (filters.since) {
    transformed.since = {
      operator: "since",
      values: [transform("since", filters.since.toString())],
    };
  }
  Object.entries(filters).forEach(([field, value]) => {
    if (value === null) {
      return;
    }

    if (Array.isArray(value)) {
      transformed[field] = {
        operator: value[0]?.operator || "is",
        values: value.map((v) => transform(field, v.value.toString())),
      };
    } else {
      transformed[field] = {
        operator: "is",
        values: [transform(field, value.toString())],
      };
    }
  });
  return transformed;
};

export const defaultGetIcon = (field: string): React.ReactNode => {
  const Icon = iconsPerField[field] || ChartActivity2;
  return React.createElement(Icon, { size: "md-regular", className: "justify-center" });
};

export const defaultFieldsToTruncate = [
  "users",
  "workspaceId",
  "keyId",
  "apiId",
  "requestId",
  "responseId",
] as const;

export function defaultShouldTruncateRow(field: string): boolean {
  return defaultFieldsToTruncate.includes(field as (typeof defaultFieldsToTruncate)[number]);
}
