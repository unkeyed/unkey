import type {
  AuditLogsFilterField,
  AuditLogsFilterUrlValue,
  AuditLogsFilterValue,
  QuerySearchParams,
} from "@/app/(app)/audit/filters.schema";
import { useCallback, useEffect } from "react";

import { useFilters } from "./use-filters";


export type SavedFiltersGroup<T> = {
  id: string;
  createdAt: number;
  filters: T;
  bookmarked?: boolean;
};

export const useBookmarkedFilters = ({ localStorageName }: { localStorageName: string }) => {
  const { filters, updateFilters } = useFilters();

  useEffect(() => {
    if (!filters.length) {
      return;
    }

    // Group current filters
    const currentGroup = filters.reduce(
      (acc, filter) => {
        let { field, value, operator } = filter;
        value = value.toString();

        if (["startTime", "endTime", "since"].includes(field)) {
          //@ts-expect-error fix later
          acc[field] = value;
        } else {
          if (!acc[field]) {
            //@ts-expect-error fix later
            acc[field] = [];
          }
          //@ts-expect-error fix later
          acc[field].push({ value, operator });
        }
        return acc;
      },
      {} as SavedFiltersGroup<QuerySearchParams>["filters"],
    );

    // Get existing saved filters
    const savedFilters = JSON.parse(
      localStorage.getItem(localStorageName) || "[]",
    ) as SavedFiltersGroup<QuerySearchParams>[];

    // Check if this combination already exists
    const filterKey = JSON.stringify(Object.entries(currentGroup).sort());
    const exists = savedFilters.some(
      (saved) => JSON.stringify(Object.entries(saved.filters).sort()) === filterKey,
    );

    if (!exists) {
      const oldWithoutBookmark = savedFilters.filter((saved) => !saved.bookmarked).slice(0, 9);
      const oldWithBookmark = savedFilters.filter((saved) => saved.bookmarked);
      const newSavedFilters = [
        ...oldWithoutBookmark,
        ...oldWithBookmark,
        {
          id: crypto.randomUUID(),
          createdAt: Date.now(),
          filters: currentGroup,
          bookmarked: false,
        },
      ].sort((a, b) => b.createdAt - a.createdAt);

      localStorage.setItem(localStorageName, JSON.stringify(newSavedFilters));
    }
  }, [filters, localStorageName]);

  const applyFilterGroup = useCallback(
    (savedGroup: SavedFiltersGroup<QuerySearchParams>) => {
      const reconstructedFilters: AuditLogsFilterValue[] = [];

      Object.entries(savedGroup.filters).forEach(([field, value]) => {
        if (["startTime", "endTime", "since"].includes(field)) {
          value &&
            reconstructedFilters.push({
              id: crypto.randomUUID(),
              field: field as AuditLogsFilterField,
              operator: "is",
              value: value as number | string,
            });
        } else {
          value !== null &&
            value !== undefined &&
            (value as AuditLogsFilterValue[]).forEach((filter) => {
              reconstructedFilters.push({
                id: crypto.randomUUID(),
                field: field as AuditLogsFilterField,
                operator: filter.operator,
                value: filter.value,
              });
            });
        }
      });

      updateFilters(reconstructedFilters);
    },
    [updateFilters],
  );
  const toggleBookmark = (groupId: string) => {
    const savedFilters = JSON.parse(
      localStorage.getItem(localStorageName) || "[]",
    ) as SavedFiltersGroup<QuerySearchParams>[];
    const updatedFilters = savedFilters.map((filter) => {
      if (filter.id === groupId) {
        return {
          ...filter,
          bookmarked: !filter.bookmarked,
        };
      }
      return filter;
    });
    localStorage.setItem(localStorageName, JSON.stringify(updatedFilters) || "[]");
    return updatedFilters;
  };

  return {
    savedFilters: (
      JSON.parse(localStorage.getItem(localStorageName) || "[]") as SavedFiltersGroup<QuerySearchParams>[]
    ).map((filter) => ({
      ...filter,
      filters: {
        ...filter.filters,
        startTime: Number.isNaN(Number(filter.filters.startTime))
          ? null
          : Number(filter.filters.startTime),
        endTime: Number.isNaN(Number(filter.filters.endTime))
          ? null
          : Number(filter.filters.endTime),
      },
    })),
    applyFilterGroup,
    toggleBookmark,
  };
};
