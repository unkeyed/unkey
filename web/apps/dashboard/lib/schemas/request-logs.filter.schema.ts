import { z } from "zod";
import type { FilterValue } from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { logsFilterFieldConfig, logsFilterFieldEnum } from "@/lib/schemas/logs.filter.schema";

export const requestLogsFilterOperatorEnum = z.enum(["is", "contains", "startsWith"]);
export type RequestLogsFilterOperator = z.infer<typeof requestLogsFilterOperatorEnum>;

const EXACT_OPERATOR = ["is"] as const;

export const requestLogsFilterFieldConfig = {
  status: { ...logsFilterFieldConfig.status, operators: EXACT_OPERATOR },
  methods: { ...logsFilterFieldConfig.methods, operators: EXACT_OPERATOR },
  paths: {
    type: "string",
    operators: ["is", "startsWith", "contains"],
  },
  host: { ...logsFilterFieldConfig.host, operators: EXACT_OPERATOR },
  requestId: { ...logsFilterFieldConfig.requestId, operators: EXACT_OPERATOR },
  appId: { ...logsFilterFieldConfig.appId, operators: EXACT_OPERATOR },
  deploymentId: { ...logsFilterFieldConfig.deploymentId, operators: EXACT_OPERATOR },
  environmentId: { ...logsFilterFieldConfig.environmentId, operators: EXACT_OPERATOR },
  startTime: { ...logsFilterFieldConfig.startTime, operators: EXACT_OPERATOR },
  endTime: { ...logsFilterFieldConfig.endTime, operators: EXACT_OPERATOR },
  since: { ...logsFilterFieldConfig.since, operators: EXACT_OPERATOR },
  region: {
    type: "string",
    operators: EXACT_OPERATOR,
  },
} as const;

export const requestLogsFilterFieldEnum = z.enum([...logsFilterFieldEnum.options, "region"]);
export type RequestLogsFilterField = z.infer<typeof requestLogsFilterFieldEnum>;

export const requestLogsFilterOutputSchema = createFilterOutputSchema(
  requestLogsFilterFieldEnum,
  requestLogsFilterOperatorEnum,
  requestLogsFilterFieldConfig,
);

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
