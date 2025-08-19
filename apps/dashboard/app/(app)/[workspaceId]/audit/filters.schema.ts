import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";
export const auditLogsFilterFieldConfig: FilterFieldConfigs = {
  bucket: {
    type: "string",
    operators: ["is"],
  },
  events: {
    type: "string",
    operators: ["is"],
  },
  users: {
    type: "string",
    operators: ["is"],
  },
  rootKeys: {
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

// Schemas
export const auditLogsFilterOperatorEnum = z.enum(["is"]);
export const auditLogsFilterFieldEnum = z.enum([
  "bucket",
  "events",
  "users",
  "rootKeys",
  "startTime",
  "endTime",
  "since",
]);

export const auditFilterOutputSchema = createFilterOutputSchema(
  auditLogsFilterFieldEnum,
  auditLogsFilterOperatorEnum,
  auditLogsFilterFieldConfig,
);

// Types
export type AuditLogsFilterOperator = z.infer<typeof auditLogsFilterOperatorEnum>;
export type AuditLogsFilterField = z.infer<typeof auditLogsFilterFieldEnum>;

export type FilterFieldConfigs = {
  bucket: StringConfig<AuditLogsFilterOperator>;
  events: StringConfig<AuditLogsFilterOperator>;
  users: StringConfig<AuditLogsFilterOperator>;
  rootKeys: StringConfig<AuditLogsFilterOperator>;
  startTime: NumberConfig<AuditLogsFilterOperator>;
  endTime: NumberConfig<AuditLogsFilterOperator>;
  since: StringConfig<AuditLogsFilterOperator>;
};

export type AuditLogsFilterUrlValue = Pick<
  FilterValue<AuditLogsFilterField, AuditLogsFilterOperator>,
  "value" | "operator"
>;
export type AuditLogsFilterValue = FilterValue<AuditLogsFilterField, AuditLogsFilterOperator>;

export type QuerySearchParams = {
  events: AuditLogsFilterUrlValue[] | null;
  users: AuditLogsFilterUrlValue[] | null;
  rootKeys: AuditLogsFilterUrlValue[] | null;
  bucket: string | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};
