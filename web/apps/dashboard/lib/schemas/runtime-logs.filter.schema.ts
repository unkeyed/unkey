import { z } from "zod";
import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";

export type RuntimeLogsAttributeMatch = {
  path: string;
  value: string;
};

export function parseRuntimeLogsAttributeMatch(input: string): RuntimeLogsAttributeMatch | null {
  const separatorIndex = input.indexOf("=");
  if (separatorIndex === -1) {
    return null;
  }

  const pathSegments = input
    .slice(0, separatorIndex)
    .split(".")
    .map((segment) => segment.trim());
  const value = input.slice(separatorIndex + 1).trim();

  if (
    pathSegments.some((segment) => segment.length === 0) ||
    pathSegments.join(".").length > 512 ||
    value.length < 3 ||
    value.length > 2_048
  ) {
    return null;
  }

  return { path: pathSegments.join("."), value };
}

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
    operators: ["contains"],
  },
  attributes: {
    type: "string",
    operators: ["contains", "is"],
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
  appId: {
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
  region: {
    type: "string",
    operators: ["is"],
  },
  instanceId: {
    type: "string",
    operators: ["is"],
  },
} as const;

// Schemas
export const runtimeLogsFilterOperatorEnum = z.enum(["is", "contains"]);

export const runtimeLogsFilterFieldEnum = z.enum([
  "severity",
  "message",
  "attributes",
  "startTime",
  "endTime",
  "since",
  "appId",
  "deploymentId",
  "environmentId",
  "region",
  "instanceId",
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
  attributes: StringConfig<RuntimeLogsFilterOperator>;
  startTime: NumberConfig<RuntimeLogsFilterOperator>;
  endTime: NumberConfig<RuntimeLogsFilterOperator>;
  since: StringConfig<RuntimeLogsFilterOperator>;
  appId: StringConfig<RuntimeLogsFilterOperator>;
  deploymentId: StringConfig<RuntimeLogsFilterOperator>;
  environmentId: StringConfig<RuntimeLogsFilterOperator>;
  region: StringConfig<RuntimeLogsFilterOperator>;
  instanceId: StringConfig<RuntimeLogsFilterOperator>;
};

export type RuntimeLogsFilterUrlValue = Pick<
  FilterValue<RuntimeLogsFilterField, RuntimeLogsFilterOperator>,
  "value" | "operator"
>;
export type RuntimeLogsFilterValue = FilterValue<RuntimeLogsFilterField, RuntimeLogsFilterOperator>;

export type RuntimeLogsQuerySearchParams = {
  severity: RuntimeLogsFilterUrlValue[] | null;
  message: RuntimeLogsFilterUrlValue[] | null;
  attributes: RuntimeLogsFilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  appId: RuntimeLogsFilterUrlValue[] | null;
  deploymentId: RuntimeLogsFilterUrlValue[] | null;
  environmentId: RuntimeLogsFilterUrlValue[] | null;
  region: RuntimeLogsFilterUrlValue[] | null;
  instanceId: RuntimeLogsFilterUrlValue[] | null;
};
