import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { DEFAULT_DEPLOYMENT_STATUS_GROUPS } from "@/lib/collections/deploy/deployment-status";
import {
  type DeploymentListFilterField,
  type DeploymentListFilterOperator,
  type DeploymentListFilterUrlValue,
  type DeploymentListFilterValue,
  type DeploymentListQuerySearchParams,
  deploymentListFilterFieldConfig,
} from "../filters.schema";

const parseAsFilterValArray = parseAsFilterValueArray<DeploymentListFilterOperator>(["is"]);

export const queryParamsPayload = {
  status: parseAsFilterValArray,
  environment: parseAsFilterValArray,
  branch: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

const arrayFields = ["status", "environment", "branch"] as const;
const timeFields = ["startTime", "endTime", "since"] as const;

// The status filter starts pre-selected rather than empty, so an absent url
// param means the default selection, not "every status".
const defaultStatusParam: DeploymentListFilterUrlValue[] = DEFAULT_DEPLOYMENT_STATUS_GROUPS.map(
  (value) => ({ value, operator: "is" }),
);

const isDefaultStatusSelection = (selection: readonly { value: string | number }[] | null) =>
  selection === null ||
  (selection.length === DEFAULT_DEPLOYMENT_STATUS_GROUPS.length &&
    selection.every((item) =>
      DEFAULT_DEPLOYMENT_STATUS_GROUPS.some((group) => group === item.value),
    ));

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: DeploymentListFilterValue[] = [];

    // Handle array filters
    arrayFields.forEach((field) => {
      const selection =
        field === "status" ? (searchParams.status ?? defaultStatusParam) : searchParams[field];
      selection?.forEach((item) => {
        activeFilters.push({
          id: crypto.randomUUID(),
          field,
          operator: item.operator,
          value: item.value,
          metadata: deploymentListFilterFieldConfig[field].getColorClass
            ? {
                colorClass: deploymentListFilterFieldConfig[field].getColorClass(
                  item.value as string,
                ),
              }
            : undefined,
        });
      });
    });

    // Handle time filters
    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof DeploymentListQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as DeploymentListFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: DeploymentListFilterValue[]) => {
      const newParams: Partial<DeploymentListQuerySearchParams> = Object.fromEntries([
        ...arrayFields.map((field) => [field, null]),
        ...timeFields.map((field) => [field, null]),
      ]);

      const filterGroups = arrayFields.reduce(
        (acc, field) => {
          acc[field] = [];
          return acc;
        },
        {} as Record<(typeof arrayFields)[number], DeploymentListFilterUrlValue[]>,
      );

      newFilters.forEach((filter) => {
        if (arrayFields.includes(filter.field as (typeof arrayFields)[number])) {
          filterGroups[filter.field as (typeof arrayFields)[number]].push({
            value: filter.value as string,
            operator: filter.operator,
          });
        } else if (filter.field === "startTime" || filter.field === "endTime") {
          newParams[filter.field] = filter.value as number;
        } else if (filter.field === "since") {
          newParams.since = filter.value as string;
        }
      });

      // Set array filters
      arrayFields.forEach((field) => {
        newParams[field] = filterGroups[field].length > 0 ? filterGroups[field] : null;
      });

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

  const toggleArrayFilter = useCallback(
    (field: (typeof arrayFields)[number], value: string) => {
      const existing = filters.find((f) => f.field === field && f.value === value);
      if (existing) {
        updateFilters(filters.filter((f) => f.id !== existing.id));
      } else {
        updateFilters([...filters, { field, id: crypto.randomUUID(), operator: "is", value }]);
      }
    },
    [filters, updateFilters],
  );

  // Whether the user narrowed the list themselves, as opposed to only seeing the
  // default status selection.
  const isFiltered = useMemo(
    () =>
      !isDefaultStatusSelection(searchParams.status) ||
      arrayFields.some((field) => field !== "status" && (searchParams[field]?.length ?? 0) > 0) ||
      timeFields.some((field) => searchParams[field] !== null),
    [searchParams],
  );

  return {
    filters,
    isFiltered,
    removeFilter,
    updateFilters,
    toggleArrayFilter,
  };
};
