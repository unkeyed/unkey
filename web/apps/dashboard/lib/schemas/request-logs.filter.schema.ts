import type { FilterValue } from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";
import { z } from "zod";

export const requestLogsFilterOperatorEnum = z.enum(["is", "contains", "startsWith"]);
export type RequestLogsFilterOperator = z.infer<typeof requestLogsFilterOperatorEnum>;

export const requestLogsFilterFieldConfig = {
  status: { ...logsFilterFieldConfig.status, operators: ["is"] },
  methods: { ...logsFilterFieldConfig.methods, operators: ["is"] },
  paths: {
    type: "string",
    operators: ["is", "startsWith", "contains"],
  },
  host: { ...logsFilterFieldConfig.host, operators: ["is"] },
  requestId: { ...logsFilterFieldConfig.requestId, operators: ["is"] },
  region: {
    type: "string",
    operators: ["is"],
  },
  userAgent: {
    type: "string",
    operators: ["contains"],
  },
  appId: { ...logsFilterFieldConfig.appId, operators: ["is"] },
  deploymentId: { ...logsFilterFieldConfig.deploymentId, operators: ["is"] },
  environmentId: { ...logsFilterFieldConfig.environmentId, operators: ["is"] },
  startTime: { ...logsFilterFieldConfig.startTime, operators: ["is"] },
  endTime: { ...logsFilterFieldConfig.endTime, operators: ["is"] },
  since: { ...logsFilterFieldConfig.since, operators: ["is"] },
} as const;

export const requestLogsFilterFieldEnum = z.enum([
  "host",
  "requestId",
  "region",
  "userAgent",
  "methods",
  "paths",
  "status",
  "appId",
  "deploymentId",
  "environmentId",
  "startTime",
  "endTime",
  "since",
]);

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

export type RequestLogsQuerySearchParams = {
  status: RequestLogsFilterUrlValue[] | null;
  methods: RequestLogsFilterUrlValue[] | null;
  paths: RequestLogsFilterUrlValue[] | null;
  appId: RequestLogsFilterUrlValue[] | null;
  deploymentId: RequestLogsFilterUrlValue[] | null;
  environmentId: RequestLogsFilterUrlValue[] | null;
  host: RequestLogsFilterUrlValue[] | null;
  requestId: RequestLogsFilterUrlValue[] | null;
  region: RequestLogsFilterUrlValue[] | null;
  userAgent: RequestLogsFilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};
