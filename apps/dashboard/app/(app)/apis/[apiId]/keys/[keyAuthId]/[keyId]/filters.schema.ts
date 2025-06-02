import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { getOutcomeColor } from "../../../_overview/utils";

export const ALLOWED_OPERATOR = ["is"] as const;
export type KeyDetailsFilterOperator = z.infer<typeof keyDetailsFilterOperatorEnum>;

export const keyDetailsFilterFieldConfig: FilterFieldConfigs = {
  startTime: {
    type: "number",
    operators: ALLOWED_OPERATOR,
  },
  endTime: {
    type: "number",
    operators: ALLOWED_OPERATOR,
  },
  since: {
    type: "string",
    operators: ALLOWED_OPERATOR,
  },
  outcomes: {
    type: "string",
    operators: ALLOWED_OPERATOR,
    validValues: KEY_VERIFICATION_OUTCOMES,
    getColorClass: getOutcomeColor,
  } as const,
};

export const keyDetailsFilterOperatorEnum = z.enum(ALLOWED_OPERATOR);
export const keyDetailsFilterFieldEnum = z.enum(["startTime", "endTime", "since", "outcomes"]);

export const filterOutputSchema = createFilterOutputSchema(
  keyDetailsFilterFieldEnum,
  keyDetailsFilterOperatorEnum,
  keyDetailsFilterFieldConfig,
);

export type KeyDetailsFilterField = z.infer<typeof keyDetailsFilterFieldEnum>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<KeyDetailsFilterOperator>;
  endTime: NumberConfig<KeyDetailsFilterOperator>;
  since: StringConfig<KeyDetailsFilterOperator>;
  outcomes: StringConfig<KeyDetailsFilterOperator>;
};

export type IsOnlyUrlValue = {
  value: string | number;
  operator: KeyDetailsFilterOperator;
};

export type KeyDetailsFilterUrlValue = Pick<
  FilterValue<KeyDetailsFilterField, KeyDetailsFilterOperator>,
  "value" | "operator"
>;

export type KeyDetailsFilterValue = FilterValue<KeyDetailsFilterField, KeyDetailsFilterOperator>;

export type KeysQuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  outcomes: IsOnlyUrlValue[] | null;
};
