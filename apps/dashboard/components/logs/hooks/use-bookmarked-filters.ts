import { type QuerySearchParams, logsFilterFieldConfig } from "@/app/(app)/logs/filters.schema";
import type { IconProps } from "@unkey/icons/src/props";
import { FileDiff } from "lucide-react";
import { useCallback, useEffect } from "react";
import { defaultFormatValue } from "../control-cloud/utils";
import { iconsPerField } from "../queries/utils";
import { parseValue } from "../queries/utils";
import type { FilterValue } from "../validation/filter.types";
export type SavedFiltersGroup<T> = {
  id: string;
  createdAt: number;
  filters: T;
  bookmarked: boolean;
};

export type ParsedSavedFiltersType = {
  id: string;
  createdAt: number;
  filters: Record<
    string,
    { operator: string; values: { value: string; color: string }[]; icon: React.FC<IconProps> }
  >;
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

  function toggleBookmark(groupId: string) {
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
  }

  const parseSavedFilters = (): ParsedSavedFiltersType[] => {
    const parsedGroups: ParsedSavedFiltersType[] = [];

    const savedFilters = (
      JSON.parse(
        localStorage.getItem(localStorageName) || "[]",
      ) as SavedFiltersGroup<QuerySearchParams>[]
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
    }));
    // Create a formatted version of each filter entry

    savedFilters.forEach((group) => {
      const formatted: Record<
        string,
        { operator: string; values: { value: string; color: string }[]; icon: React.FC<IconProps> }
      > = {};

      // Handle time filters
      if (group.filters.startTime || group.filters.endTime || group.filters.since) {
        formatted.time = {
          operator: "",
          values: [],
          icon: iconsPerField.time,
        };

        if (group.filters.startTime && group.filters.endTime) {
          formatted.time.operator = "between";
          formatted.time.values.push({
            value: defaultFormatValue(Number(group.filters.startTime), "startTime"),
            color: "",
          });
          formatted.time.values.push({
            value: defaultFormatValue(Number(group.filters.endTime), "endTime"),
            color: "",
          });
        } else if (group.filters.startTime) {
          formatted.time.operator = "starts from";
          formatted.time.values.push({
            value: defaultFormatValue(Number(group.filters.startTime), "startTime"),
            color: "",
          });
        } else if (group.filters.since) {
          formatted.time.operator = "since";
          formatted.time.values.push({
            value: String(group.filters.since),
            color: "",
          });
        }
      }

      // Handle regular filters
      Object.entries(group.filters).forEach(([field, value]) => {
        if (["startTime", "endTime", "since"].includes(field)) {
          return;
        }

        if (Array.isArray(value)) {
          value.forEach((filter) => {
            const formattedValue = parseValue(filter.value.toString());

            if (!formatted[field]) {
              formatted[field] = {
                operator: filter.operator,
                values: [{ value: formattedValue.phrase || "", color: formattedValue.color || "" }],
                icon: iconsPerField[field] || FileDiff,
              };
            } else {
              formatted[field].values.push({
                value: formattedValue.phrase,
                color: formattedValue.color || "",
              });
            }
          });
        }
      });

      parsedGroups.push({
        id: group.id,
        createdAt: group.createdAt,
        bookmarked: group.bookmarked,
        filters: formatted,
      });
    });

    return parsedGroups || [];
  };

  return {
    savedFilters: (
      JSON.parse(
        localStorage.getItem(localStorageName) || "[]",
      ) as SavedFiltersGroup<QuerySearchParams>[]
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
    parseSavedFilters,
    applyFilterGroup,
    toggleBookmark,
  };
}
