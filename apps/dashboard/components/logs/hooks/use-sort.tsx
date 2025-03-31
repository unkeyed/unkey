import {
  type SortDirection,
  type SortUrlValue,
  parseAsSortArray,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { useQueryState } from "nuqs";
import { useCallback } from "react";
import type { z } from "zod";

export type SortField<TSortFields> = TSortFields extends z.ZodEnum<infer U> ? U : TSortFields;

export function createSortArrayParser<TSortFields extends string>() {
  return parseAsSortArray<TSortFields>();
}

export function useSort<TSortFields extends string>(paramName = "sorts") {
  const [sortParams, setSortParams] = useQueryState(
    paramName,
    createSortArrayParser<TSortFields>(),
  );

  const getSortDirection = useCallback(
    (columnKey: TSortFields): SortDirection | undefined => {
      return sortParams?.find((sort) => sort.column === columnKey)?.direction;
    },
    [sortParams],
  );

  const toggleSort = useCallback(
    (columnKey: TSortFields, multiSort = false, order: "asc" | "desc" = "desc") => {
      const currentSort = sortParams?.find((sort) => sort.column === columnKey);
      const otherSorts = sortParams?.filter((sort) => sort.column !== columnKey) ?? [];
      let newSorts: SortUrlValue<TSortFields>[];

      if (!currentSort) {
        // Add new sort
        newSorts = multiSort
          ? [...(sortParams ?? []), { column: columnKey, direction: order }]
          : [{ column: columnKey, direction: order }];
      } else if (currentSort.direction === "asc") {
        // Toggle to desc
        newSorts = multiSort
          ? [...otherSorts, { column: columnKey, direction: "desc" }]
          : [{ column: columnKey, direction: "desc" }];
      } else {
        // Remove sort
        newSorts = multiSort ? otherSorts : [];
      }

      setSortParams(newSorts.length > 0 ? newSorts : null);
    },
    [sortParams, setSortParams],
  );

  return {
    sorts: sortParams ?? [],
    getSortDirection,
    toggleSort,
  } as const;
}
