import { type Parser, parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";

export const STATUSES = [200, 400, 500] as const;
export const METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"] as const;

export type ResponseStatus = (typeof STATUSES)[number];
export type HttpMethod = (typeof METHODS)[number];

type FilterUrlValue = {
  value: string | number;
  operator: FilterOperator;
};

export type QuerySearchParams = {
  host: FilterUrlValue | null;
  requestId: FilterUrlValue | null;
  methods: FilterUrlValue[] | null;
  paths: FilterUrlValue[] | null;
  status: FilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
};

export type FilterField = keyof QuerySearchParams;
export type FilterOperator = "is" | "contains" | "startsWith" | "endsWith";

export interface FilterValue {
  id: string;
  field: FilterField;
  operator: FilterOperator;
  value: string | number | ResponseStatus | HttpMethod;
  metadata?: {
    colorClass?: string;
    icon?: React.ReactNode;
  };
}

const parseAsFilterValue: Parser<FilterUrlValue | null> = {
  parse: (str: string | null) => {
    if (!str) {
      return null;
    }
    try {
      // Format: operator:value (e.g., "is:200" for {operator: "is", value: "200"})
      const [operator, val] = str.split(/:(.+)/);
      if (!operator || !val) {
        return null;
      }

      if (!["is", "contains", "startsWith", "endsWith"].includes(operator)) {
        return null;
      }

      return {
        operator: operator as FilterOperator,
        value: val,
      };
    } catch {
      return null;
    }
  },
  serialize: (value: FilterUrlValue | null) => {
    if (!value) {
      return "";
    }
    return `${value.operator}:${value.value}`;
  },
};

const parseAsFilterValueArray: Parser<FilterUrlValue[]> = {
  parse: (str: string | null) => {
    if (!str) {
      return [];
    }
    try {
      // Format: operator:value,operator:value (e.g., "is:200,is:404")
      return str.split(",").map((item) => {
        const [operator, val] = item.split(/:(.+)/);
        if (!["is", "contains", "startsWith", "endsWith"].includes(operator)) {
          throw new Error("Invalid operator");
        }
        return {
          operator: operator as FilterOperator,
          value: val,
        };
      });
    } catch {
      return [];
    }
  },
  serialize: (value: FilterUrlValue[]) => {
    if (!value?.length) {
      return "";
    }
    return value.map((v) => `${v.operator}:${v.value}`).join(",");
  },
};

export const queryParamsPayload = {
  requestId: parseAsFilterValue,
  host: parseAsFilterValue,
  methods: parseAsFilterValueArray,
  paths: parseAsFilterValueArray,
  status: parseAsFilterValueArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
} as const;

export const filterFieldConfig = {
  status: {
    type: "number",
    operators: ["is"] as const,
    getColorClass: (value: number) => {
      if (value >= 500) {
        return "bg-error-9";
      }
      if (value >= 400) {
        return "bg-warning-8";
      }
      return "bg-success-9";
    },
  },
  methods: {
    type: "string",
    operators: ["is"] as const,
    validValues: METHODS,
  },
  paths: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"] as const,
  },
  host: {
    type: "string",
    operators: ["is", "contains"] as const,
  },
  requestId: {
    type: "string",
    operators: ["is"] as const,
  },
  startTime: {
    type: "number",
    operators: ["is"] as const,
  },
  endTime: {
    type: "number",
    operators: ["is"] as const,
  },
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload);

  const filters = useMemo(() => {
    const activeFilters: FilterValue[] = [];

    searchParams.status?.forEach((status) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "status",
        operator: status.operator,
        value: status.value as ResponseStatus,
        metadata: {
          colorClass: filterFieldConfig.status.getColorClass(status.value as number),
        },
      });
    });

    searchParams.methods?.forEach((method) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "methods",
        operator: method.operator,
        value: method.value as HttpMethod,
      });
    });

    searchParams.paths?.forEach((pathFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "paths",
        operator: pathFilter.operator,
        value: pathFilter.value,
      });
    });

    if (searchParams.host) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "host",
        operator: searchParams.host.operator,
        value: searchParams.host.value,
      });
    }

    if (searchParams.requestId) {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "requestId",
        operator: searchParams.requestId.operator,
        value: searchParams.requestId.value,
      });
    }

    ["startTime", "endTime"].forEach((field) => {
      const value = searchParams[field as keyof QuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as FilterField,
          operator: "is",
          value: value as number,
        });
      }
    });

    console.log({ activeFilters });
    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: FilterValue[]) => {
      const newParams: Partial<QuerySearchParams> = {
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
        methods: null,
        status: null,
      };

      // Group filters by field
      const responseStatusFilters: FilterUrlValue[] = [];
      const methodFilters: FilterUrlValue[] = [];
      const pathFilters: FilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "status":
            responseStatusFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "methods":
            methodFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "paths":
            pathFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "host":
            newParams.host = {
              value: filter.value as string,
              operator: filter.operator,
            };
            break;
          case "requestId":
            newParams.requestId = {
              value: filter.value as string,
              operator: filter.operator,
            };
            break;
          case "startTime":
          case "endTime":
            newParams[filter.field] = filter.value as number;
            break;
        }
      });

      // Set arrays to null when empty, otherwise use the filtered values
      newParams.status = responseStatusFilters.length > 0 ? responseStatusFilters : null;
      newParams.methods = methodFilters.length > 0 ? methodFilters : null;
      newParams.paths = pathFilters.length > 0 ? pathFilters : null;

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

  const addFilter = useCallback(
    (
      field: FilterField,
      operator: FilterOperator,
      value: string | number | ResponseStatus | HttpMethod,
    ) => {
      const newFilter: FilterValue = {
        id: crypto.randomUUID(),
        field,
        operator,
        value,
        metadata:
          field === "status"
            ? {
                colorClass: filterFieldConfig.status.getColorClass(value as number),
              }
            : undefined,
      };

      updateFilters([...filters, newFilter]);
    },
    [filters, updateFilters],
  );

  return {
    filters,
    addFilter,
    removeFilter,
    updateFilters,
  };
};
