import type { FilterValue } from "@/components/logs/validation/filter.types";
import { logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";

export type RequestLogsFilterOperator = "is" | "contains" | "startsWith";

export const requestLogsFilterFieldConfig = {
  ...logsFilterFieldConfig,
  paths: {
    type: "string",
    operators: ["is", "startsWith", "contains"],
  },
  region: {
    type: "string",
    operators: ["is"],
  },
} as const;

export type RequestLogsFilterField = keyof typeof requestLogsFilterFieldConfig;

export type RequestLogsFilterUrlValue = Pick<
  FilterValue<RequestLogsFilterField, RequestLogsFilterOperator>,
  "value" | "operator"
>;
export type RequestLogsFilterValue = FilterValue<RequestLogsFilterField, RequestLogsFilterOperator>;

type RequestLogsArrayField = Exclude<RequestLogsFilterField, "startTime" | "endTime" | "since">;

export type RequestLogsQuerySearchParams = Record<
  RequestLogsArrayField,
  RequestLogsFilterUrlValue[] | null
> & {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};
