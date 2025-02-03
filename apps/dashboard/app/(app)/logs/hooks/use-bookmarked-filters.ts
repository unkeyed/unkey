import { useCallback, useEffect } from "react";
import { filterFieldConfig } from "../filters.schema";
import type { FilterField, FilterUrlValue, FilterValue } from "../filters.type";
import { useFilters } from "./use-filters";

export type SavedFiltersGroup = {
  id: string;
  createdAt: number;
  filters: {
    status?: FilterUrlValue[];
    methods?: FilterUrlValue[];
    paths?: FilterUrlValue[];
    host?: FilterUrlValue[];
    requestId?: FilterUrlValue[];
    startTime?: number;
    endTime?: number;
    since?: string;
  };
};

export const useBookmarkedFilters = () => {
  const { filters, updateFilters } = useFilters();

  useEffect(() => {
    if (!filters.length) {
      return;
    }

    // Group current filters
    const currentGroup = filters.reduce(
      (acc, filter) => {
        const { field, value, operator } = filter;

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
      {} as SavedFiltersGroup["filters"],
    );

    // Get existing saved filters
    const savedFilters = JSON.parse(
      localStorage.getItem("savedFilters") || "[]",
    ) as SavedFiltersGroup[];

    // Check if this combination already exists
    const filterKey = JSON.stringify(Object.entries(currentGroup).sort());
    const exists = savedFilters.some(
      (saved) => JSON.stringify(Object.entries(saved.filters).sort()) === filterKey,
    );

    if (!exists) {
      const newSavedFilters = [
        ...savedFilters,
        {
          id: crypto.randomUUID(),
          createdAt: Date.now(),
          filters: currentGroup,
        },
      ].slice(-5); // Keep last 5 unique combinations

      localStorage.setItem("savedFilters", JSON.stringify(newSavedFilters));
    }
  }, [filters]);

  const applyFilterGroup = useCallback(
    (savedGroup: SavedFiltersGroup) => {
      const reconstructedFilters: FilterValue[] = [];

      Object.entries(savedGroup.filters).forEach(([field, value]) => {
        if (["startTime", "endTime", "since"].includes(field)) {
          reconstructedFilters.push({
            id: crypto.randomUUID(),
            field: field as FilterField,
            operator: "is",
            value: value as number | string,
          });
        } else {
          (value as FilterUrlValue[]).forEach((filter) => {
            reconstructedFilters.push({
              id: crypto.randomUUID(),
              field: field as FilterField,
              operator: filter.operator,
              value: filter.value,
              metadata:
                field === "status"
                  ? {
                      colorClass: filterFieldConfig.status.getColorClass?.(filter.value as number),
                    }
                  : undefined,
            });
          });
        }
      });

      updateFilters(reconstructedFilters);
    },
    [updateFilters],
  );

  return {
    savedFilters: JSON.parse(localStorage.getItem("savedFilters") || "[]") as SavedFiltersGroup[],
    applyFilterGroup,
  };
};
