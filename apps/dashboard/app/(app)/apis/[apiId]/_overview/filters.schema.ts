import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { getOutcomeColor } from "./utils";

export const keysOverviewFilterFieldConfig: FilterFieldConfigs = {
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
  keyIds: {
    type: "string",
    operators: ["is", "contains"],
  },
  names: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
  identities: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
  outcomes: {
    type: "string",
    operators: ["is"],
    validValues: KEY_VERIFICATION_OUTCOMES,
    getColorClass: getOutcomeColor,
  } as const,
};

// Schemas
export const keysOverviewFilterOperatorEnum = z.enum(["is", "contains", "startsWith", "endsWith"]);

export const keysOverviewFilterFieldEnum = z.enum([
  "startTime",
  "endTime",
  "since",
  "keyIds",
  "names",
  "outcomes",
  "identities",
]);

export const filterOutputSchema = createFilterOutputSchema(
  keysOverviewFilterFieldEnum,
  keysOverviewFilterOperatorEnum,
  keysOverviewFilterFieldConfig,
);

// Types
export type KeysOverviewFilterOperator = z.infer<typeof keysOverviewFilterOperatorEnum>;

export type KeysOverviewFilterField = z.infer<typeof keysOverviewFilterFieldEnum>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<KeysOverviewFilterOperator>;
  endTime: NumberConfig<KeysOverviewFilterOperator>;
  since: StringConfig<KeysOverviewFilterOperator>;
  keyIds: StringConfig<KeysOverviewFilterOperator>;
  names: StringConfig<KeysOverviewFilterOperator>;
  outcomes: StringConfig<KeysOverviewFilterOperator>;
  identities: StringConfig<KeysOverviewFilterOperator>;
};

export type IsOnlyUrlValue = {
  value: string | number;
  operator: "is";
};

export type IsContainsUrlValue = {
  value: string;
  operator: "is" | "contains";
};

export type AllOperatorsUrlValue = {
  value: string;
  operator: "is" | "contains" | "startsWith" | "endsWith";
};

export type KeysOverviewFilterUrlValue = Pick<
  FilterValue<KeysOverviewFilterField, KeysOverviewFilterOperator>,
  "value" | "operator"
>;

export type KeysOverviewFilterValue = FilterValue<
  KeysOverviewFilterField,
  KeysOverviewFilterOperator
>;

export type KeysQuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  keyIds: IsContainsUrlValue[] | null;
  names: AllOperatorsUrlValue[] | null;
  outcomes: IsOnlyUrlValue[] | null;
  identities: AllOperatorsUrlValue[] | null;
};
