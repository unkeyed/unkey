import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

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
    validValues: ["GET", "POST", "PUT", "DELETE", "PATCH"] as const,
  },
  paths: {
    type: "string",
    operators: ["contains"],
  },
  host: {
    type: "string",
    operators: ["is"],
  },
  deploymentId: {
    type: "string",
    operators: ["contains"],
  },
  environmentId: {
    type: "string",
    operators: ["contains"],
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
export const logsFilterOperatorEnum = z.enum(["is", "contains"]);

export const logsFilterFieldEnum = z.enum([
  "methods",
  "paths",
  "status",
  "host",
  "deploymentId",
  "environmentId",
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
  deploymentId: StringConfig<LogsFilterOperator>;
  environmentId: StringConfig<LogsFilterOperator>;
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
  host: LogsFilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};
