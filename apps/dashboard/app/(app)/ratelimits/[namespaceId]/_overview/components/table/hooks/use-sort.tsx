import {
  type SortDirection,
  type SortUrlValue,
  parseAsSortArray,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { useQueryState } from "nuqs";
import { useCallback } from "react";
import type { z } from "zod";
import type { sortFields } from "../query-logs.schema";

type SortField = z.infer<typeof sortFields>;

export function useSort() {
  const [sortParams, setSortParams] = useQueryState("sorts", parseAsSortArray<SortField>());

  const getSortDirection = useCallback(
    (columnKey: SortField): SortDirection | undefined => {
      return sortParams?.find((sort) => sort.column === columnKey)?.direction;
    },
    [sortParams],
  );

  const toggleSort = useCallback(
    (columnKey: SortField, multiSort = false) => {
      const currentSort = sortParams?.find((sort) => sort.column === columnKey);
      const otherSorts = sortParams?.filter((sort) => sort.column !== columnKey) ?? [];

      let newSorts: SortUrlValue<SortField>[];

      if (!currentSort) {
        // Add new sort
        newSorts = multiSort
          ? [...(sortParams ?? []), { column: columnKey, direction: "asc" }]
          : [{ column: columnKey, direction: "asc" }];
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
