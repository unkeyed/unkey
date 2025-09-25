import {
  type QuerySearchParams,
  logsFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/logs/filters.schema";
import { isBrowser } from "@/lib/utils";
import { useCallback, useEffect, useState } from "react";
import type { FilterValue } from "../validation/filter.types";

export type SavedFiltersGroup<T> = {
  id: string;
  createdAt: number;
  filters: T;
  bookmarked: boolean;
};

type BookmarkFilterType<T extends FilterValue> = {
  localStorageName: string;
  filters: T[];
  updateFilters: (filters: T[]) => void;
};

export function useBookmarkedFilters<T extends FilterValue>({
  localStorageName,
  filters,
  updateFilters,
}: BookmarkFilterType<T>) {
  const [savedFilters, setSavedFilters] = useState<SavedFiltersGroup<QuerySearchParams>[]>([]);

  useEffect(() => {
    const loadedFilters = JSON.parse(
      getLocalStorage(localStorageName),
    ) as SavedFiltersGroup<QuerySearchParams>[];
    setSavedFilters(loadedFilters);
  }, [localStorageName]);

  useEffect(() => {
    if (!filters.length || !isBrowser) {
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
          //@ts-expect-error fix later
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
      getLocalStorage(localStorageName),
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

      setLocalStorage(localStorageName, JSON.stringify(newSavedFilters));
      setSavedFilters(newSavedFilters);
    }
  }, [filters, localStorageName]);

  const applyFilterGroup = useCallback(
    (savedGroup: SavedFiltersGroup<QuerySearchParams>) => {
      const reconstructedFilters: T[] = Object.entries(savedGroup.filters).flatMap(
        ([field, value]) => {
          if (!value) {
            return [] as T[];
          }

          if (["startTime", "endTime", "since"].includes(field)) {
            return [
              {
                id: crypto.randomUUID(),
                field: field,
                operator: "is",
                value: value as number | string,
                metadata: undefined,
              } as T,
            ];
          }

          if (!Array.isArray(value)) {
            return [] as T[];
          }

          return value.map((filter) => {
            return {
              id: crypto.randomUUID(),
              field: field,
              operator: filter.operator,
              value: filter.value,
              metadata:
                field === "status"
                  ? {
                      colorClass: logsFilterFieldConfig.status.getColorClass?.(
                        filter.value as number,
                      ),
                    }
                  : undefined,
            } as T;
          });
        },
      );

      updateFilters(reconstructedFilters);
    },
    [updateFilters],
  );

  const toggleBookmark = useCallback(
    (groupId: string) => {
      if (!isBrowser) {
        return;
      }

      const currentSavedFilters = JSON.parse(
        getLocalStorage(localStorageName),
      ) as SavedFiltersGroup<QuerySearchParams>[];

      const updatedFilters = currentSavedFilters.map((filter) => {
        if (filter.id === groupId) {
          return {
            ...filter,
            bookmarked: !filter.bookmarked,
          };
        }
        return filter;
      });

      setLocalStorage(localStorageName, JSON.stringify(updatedFilters));
      setSavedFilters(updatedFilters);
    },
    [localStorageName],
  );

  const processedFilters = savedFilters.map((filter) => ({
    ...filter,
    filters: {
      ...filter.filters,
      startTime: Number.isNaN(Number(filter.filters.startTime))
        ? null
        : Number(filter.filters.startTime),
      endTime: Number.isNaN(Number(filter.filters.endTime)) ? null : Number(filter.filters.endTime),
    },
  }));

  return {
    savedFilters: processedFilters,
    applyFilterGroup,
    toggleBookmark,
  };
}

const getLocalStorage = (key: string) => {
  if (!isBrowser) {
    return "[]";
  }
  return localStorage.getItem(key) || "[]";
};

const setLocalStorage = (key: string, value: string) => {
  if (!isBrowser) {
    return;
  }
  localStorage.setItem(key, value);
};
