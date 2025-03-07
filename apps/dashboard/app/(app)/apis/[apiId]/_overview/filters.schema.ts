import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

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
    operators: ["is", "contains"],
  },
  outcomes: {
    type: "string",
    operators: ["is"],
    validValues: [
      "VALID",
      "INSUFFICIENT_PERMISSIONS",
      "RATE_LIMITED",
      "FORBIDDEN",
      "DISABLED",
      "EXPIRED",
      "USAGE_EXCEEDED",
      "",
    ],
    getColorClass: (value) => {
      switch (value) {
        case "VALID":
          return "bg-success-9";
        case "RATE_LIMITED":
          return "bg-warning-9";
        case "INSUFFICIENT_PERMISSIONS":
        case "FORBIDDEN":
          return "bg-error-9";
        case "DISABLED":
          return "bg-gray-9";
        case "EXPIRED":
          return "bg-orange-9";
        case "USAGE_EXCEEDED":
          return "bg-feature-9";
        default:
          return "bg-gray-5";
      }
    },
  } as const,
};

// Schemas
export const keysOverviewFilterOperatorEnum = z.enum(["is", "contains"]);
export const keysOverviewFilterFieldEnum = z.enum([
  "startTime",
  "endTime",
  "since",
  "keyIds",
  "names",
  "outcomes",
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
  keyIds: KeysOverviewFilterUrlValue[] | null;
  names: KeysOverviewFilterUrlValue[] | null;
  outcomes: KeysOverviewFilterUrlValue[] | null;
};
