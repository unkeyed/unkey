import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

// Configuration
export const runtimeLogsFilterFieldConfig: RuntimeLogsFilterFieldConfigs = {
  severity: {
    type: "string",
    operators: ["is"],
    validValues: ["ERROR", "WARN", "INFO", "DEBUG"] as const,
    getColorClass: (value) => {
      const colors: Record<string, string> = {
        ERROR: "text-error-11 bg-error-9",
        WARN: "text-warning-11 bg-warning-8",
        INFO: "text-info-11 bg-info-9",
        DEBUG: "text-grayA-9 bg-grayA-9",
      };
      return colors[value.toUpperCase()] || colors.DEBUG;
    },
  },
  message: {
    type: "string",
    operators: ["is", "contains"],
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
  deploymentId: {
    type: "string",
    operators: ["is"],
  },
  environmentId: {
    type: "string",
    operators: ["is"],
  },
} as const;

// Schemas
export const runtimeLogsFilterOperatorEnum = z.enum(["is", "contains"]);

export const runtimeLogsFilterFieldEnum = z.enum([
  "severity",
  "message",
  "startTime",
  "endTime",
  "since",
  "deploymentId",
  "environmentId",
]);

export const runtimeLogsFilterOutputSchema = createFilterOutputSchema(
  runtimeLogsFilterFieldEnum,
  runtimeLogsFilterOperatorEnum,
  runtimeLogsFilterFieldConfig,
);

// Types
export type RuntimeLogsFilterOperator = z.infer<typeof runtimeLogsFilterOperatorEnum>;
export type RuntimeLogsFilterField = z.infer<typeof runtimeLogsFilterFieldEnum>;

export type RuntimeLogsFilterFieldConfigs = {
  severity: StringConfig<RuntimeLogsFilterOperator>;
  message: StringConfig<RuntimeLogsFilterOperator>;
  startTime: NumberConfig<RuntimeLogsFilterOperator>;
  endTime: NumberConfig<RuntimeLogsFilterOperator>;
  since: StringConfig<RuntimeLogsFilterOperator>;
  deploymentId: StringConfig<RuntimeLogsFilterOperator>;
  environmentId: StringConfig<RuntimeLogsFilterOperator>;
};

export type RuntimeLogsFilterUrlValue = Pick<
  FilterValue<RuntimeLogsFilterField, RuntimeLogsFilterOperator>,
  "value" | "operator"
>;
export type RuntimeLogsFilterValue = FilterValue<RuntimeLogsFilterField, RuntimeLogsFilterOperator>;

export type RuntimeLogsQuerySearchParams = {
  severity: RuntimeLogsFilterUrlValue[] | null;
  message: RuntimeLogsFilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  deploymentId: RuntimeLogsFilterUrlValue[] | null;
  environmentId: RuntimeLogsFilterUrlValue[] | null;
};
