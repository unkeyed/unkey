import { METHODS } from "./constants";

import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

// Configuration
export const logsFilterFieldConfig: FilterFieldConfigs = {
  status: {
    type: "number",
    operators: ["is"],
    getColorClass: (value) => {
      if (value >= 500) {
        return "bg-error-9";
      }
      if (value >= 400) {
        return "bg-warning-8";
      }
      return "bg-success-9";
    },
    validate: (value) => value >= 200 && value <= 599,
  },
  methods: {
    type: "string",
    operators: ["is"],
    validValues: METHODS,
  },
  paths: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
  host: {
    type: "string",
    operators: ["is"],
  },
  requestId: {
    type: "string",
    operators: ["is"],
  },
  startTime: {
    type: "number",
    operators: ["is"],
  },
  endTime: {
    type: "number",
    operators: ["is"],
  },
  since: {
    type: "string",
    operators: ["is"],
  },
} as const;

export interface StatusConfig extends NumberConfig {
  type: "number";
  operators: ["is"];
  validate: (value: number) => boolean;
}

// Schemas
export const logsFilterOperatorEnum = z.enum(["is", "contains", "startsWith", "endsWith"]);

export const logsFilterFieldEnum = z.enum([
  "host",
  "requestId",
  "methods",
  "paths",
  "status",
  "startTime",
  "endTime",
  "since",
]);

export const filterOutputSchema = createFilterOutputSchema(
  logsFilterFieldEnum,
  logsFilterOperatorEnum,
  logsFilterFieldConfig,
);

// Types
export type LogsFilterOperator = z.infer<typeof logsFilterOperatorEnum>;
export type LogsFilterField = z.infer<typeof logsFilterFieldEnum>;

export type FilterFieldConfigs = {
  status: StatusConfig;
  methods: StringConfig<LogsFilterOperator>;
  paths: StringConfig<LogsFilterOperator>;
  host: StringConfig<LogsFilterOperator>;
  requestId: StringConfig<LogsFilterOperator>;
  startTime: NumberConfig<LogsFilterOperator>;
  endTime: NumberConfig<LogsFilterOperator>;
  since: StringConfig<LogsFilterOperator>;
};

export type LogsFilterUrlValue = Pick<
  FilterValue<LogsFilterField, LogsFilterOperator>,
  "value" | "operator"
>;
export type LogsFilterValue = FilterValue<LogsFilterField, LogsFilterOperator>;

export type QuerySearchParams = {
  methods: LogsFilterUrlValue[] | null;
  paths: LogsFilterUrlValue[] | null;
  status: LogsFilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  host: LogsFilterUrlValue[] | null;
  requestId: LogsFilterUrlValue[] | null;
};
