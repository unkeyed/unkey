import { parseAsString, useQueryState } from "nuqs";
import { useMemo } from "react";
import type { RootKeysFilterValue } from "../filters.schema";

// The list only ever filters by the search box, so the URL holds a plain
// `?name=` string. The filter-value shape is kept because the shared paginated
// list hook speaks it.
export const useFilters = () => {
  const [name] = useQueryState("name", parseAsString.withDefault(""));

  const filters = useMemo<RootKeysFilterValue[]>(
    () =>
      name.trim()
        ? [{ id: "name:contains", field: "name", operator: "contains", value: name.trim() }]
        : [],
    [name],
  );

  return { filters };
};
