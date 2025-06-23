import { METHODS } from "./constants";
import {
  type BaseFieldConfig,
  COMMON_STRING_OPERATORS,
  createFilterSchema,
} from "@/lib/filter-builders";
import type { z } from "zod";

const LOGS_OPERATORS = ["is", "contains", "startsWith", "endsWith"] as const;

const LOGS_FIELDS = [
  "status",
  "methods",
  "paths",
  "host",
  "requestId",
  "startTime",
  "endTime",
  "since",
] as const;

const LOGS_FIELD_CONFIGS = {
  status: {
    type: "string" as const,
    operators: ["is"] as const,
    getColorClass: (value: unknown) => {
      const numValue = value as number;
      if (numValue >= 500) {
        return "bg-error-9";
      }
      if (numValue >= 400) {
        return "bg-warning-8";
      }
      return "bg-success-9";
    },
  },
  methods: {
    type: "string" as const,
    operators: ["is"] as const,
    validValues: METHODS,
  },
  paths: {
    type: "string" as const,
    operators: COMMON_STRING_OPERATORS,
  },
  host: {
    type: "string" as const,
    operators: ["is"] as const,
  },
  requestId: {
    type: "string" as const,
    operators: ["is"] as const,
  },
  startTime: {
    type: "number" as const,
    operators: ["is"] as const,
  },
  endTime: {
    type: "number" as const,
    operators: ["is"] as const,
  },
  since: {
    type: "string" as const,
    operators: ["is"] as const,
  },
} as const satisfies Record<
  (typeof LOGS_FIELDS)[number],
  BaseFieldConfig<readonly string[]>
>;

export const logsFilter = createFilterSchema(
  LOGS_OPERATORS,
  LOGS_FIELDS,
  LOGS_FIELD_CONFIGS
);

export const logsFilterFieldConfig = logsFilter.filterFieldConfig;
export const logsFilterOperatorEnum = logsFilter.operatorEnum;
export const logsFilterFieldEnum = logsFilter.fieldEnum;
export const filterOutputSchema = logsFilter.filterOutputSchema;

export type LogsFilterOperator = z.infer<typeof logsFilterOperatorEnum>;
export type LogsFilterField = z.infer<typeof logsFilterFieldEnum>;
export type LogsFilterValue = typeof logsFilter.types.FilterValue;
export type LogsFilterUrlValue = typeof logsFilter.types.UrlValue;
export type QuerySearchParams = typeof logsFilter.types.QuerySearchParams;
