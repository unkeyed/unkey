// ============================================================================
// LOGS FILTER CONFIGURATION
// ============================================================================

import type {
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import {
  COMMON_NUMBER_OPERATORS,
  COMMON_STRING_OPERATORS,
  createFilterSchema,
} from "@/lib/filter-builder-1";
import type { z } from "zod";

/**
 * Logs filter configuration - includes special time fields
 */

const logsFilterConfigs = {
  status: {
    type: "number" as const,
    operators: COMMON_NUMBER_OPERATORS,
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
    operators: COMMON_STRING_OPERATORS,
  },
  paths: {
    type: "string" as const,
    operators: COMMON_STRING_OPERATORS,
  },
  host: {
    type: "string" as const,
    operators: COMMON_STRING_OPERATORS,
  },
  requestId: {
    type: "string" as const,
    operators: COMMON_STRING_OPERATORS,
  },
  startTime: {
    type: "number" as const,
    operators: COMMON_NUMBER_OPERATORS,
    isTimeField: true as const,
  },
  endTime: {
    type: "number" as const,
    operators: COMMON_NUMBER_OPERATORS,
    isTimeField: true as const,
  },
  since: {
    type: "string" as const,
    operators: COMMON_STRING_OPERATORS,
    isRelativeTimeField: true as const,
  },
} as const;

// Generate logs filter schema
const logsSchema = createFilterSchema("logs", logsFilterConfigs);

export const logsFilterFieldConfig = logsFilterConfigs;
export const logsFilterOperatorEnum = logsSchema.operatorEnum;
export const logsFilterFieldEnum = logsSchema.fieldEnum;
export const filterOutputSchema = logsSchema.filterOutputSchema;
export const queryLogsPayload = logsSchema.apiQuerySchema;
export const queryParamsPayload = logsSchema.queryParamsPayload;

export type LogsFilterOperator = typeof logsSchema.types.Operator;
export type LogsFilterField = keyof typeof logsFilterConfigs;
export type LogsFilterValue = typeof logsSchema.types.FilterValue;
export type LogsFilterUrlValue = typeof logsSchema.types.AllOperatorsUrlValue;
export type QuerySearchParams = typeof logsSchema.types.QuerySearchParams;

export type QueryLogsPayload = z.infer<typeof logsSchema.apiQuerySchema>;

// Backwards compatibility
export interface StatusConfig extends NumberConfig<"is"> {
  type: "number";
  operators: ["is"];
  validate: (value: number) => boolean;
}

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
