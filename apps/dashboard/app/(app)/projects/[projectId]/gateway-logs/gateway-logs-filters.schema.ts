import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";
import { METHODS } from "./constants";

// Constants
const ALL_OPERATORS = ["is", "contains", "startsWith", "endsWith"] as const;

// Types
export type GatewayLogsFilterOperator = (typeof ALL_OPERATORS)[number];

type StatusConfig = NumberConfig<GatewayLogsFilterOperator> & {
  type: "number";
  operators: ["is"];
  getColorClass: (value: number) => string;
  validate: (value: number) => boolean;
};

type FilterFieldConfigs = {
  status: StatusConfig;
  methods: StringConfig<GatewayLogsFilterOperator>;
  paths: StringConfig<GatewayLogsFilterOperator>;
  host: StringConfig<GatewayLogsFilterOperator>;
  requestId: StringConfig<GatewayLogsFilterOperator>;
  startTime: NumberConfig<GatewayLogsFilterOperator>;
  endTime: NumberConfig<GatewayLogsFilterOperator>;
  since: StringConfig<GatewayLogsFilterOperator>;
};

// Configuration
export const gatewayLogsFilterFieldConfig: FilterFieldConfigs = {
  status: {
    type: "number",
    operators: ["is"],
    getColorClass: (value) => {
      if (value >= 500) {
        return "bg-error-9";
      }
      if (value >= 400) {
        return "bg-warning-8";
      }
      return "bg-success-9";
    },
    validate: (value) => value >= 200 && value <= 599,
  },
  methods: {
    type: "string",
    operators: ["is"],
    validValues: METHODS,
  },
  paths: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
  host: {
    type: "string",
    operators: ["is"],
  },
  requestId: {
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
export const gatewayLogsFilterOperatorEnum = z.enum(ALL_OPERATORS);
export const gatewayLogsFilterFieldEnum = z.enum([
  "status",
  "methods",
  "paths",
  "host",
  "requestId",
  "startTime",
  "endTime",
  "since",
]);

export const gatewayLogsFilterOutputSchema = createFilterOutputSchema(
  gatewayLogsFilterFieldEnum,
  gatewayLogsFilterOperatorEnum,
  gatewayLogsFilterFieldConfig,
);

// Derived types
export type GatewayLogsFilterField = z.infer<typeof gatewayLogsFilterFieldEnum>;

export type GatewayLogsFilterUrlValue = {
  value: string;
  operator: GatewayLogsFilterOperator;
};

export type GatewayLogsFilterValue = FilterValue<GatewayLogsFilterField, GatewayLogsFilterOperator>;

export type GatewayLogsQuerySearchParams = {
  status: GatewayLogsFilterUrlValue[] | null;
  methods: GatewayLogsFilterUrlValue[] | null;
  paths: GatewayLogsFilterUrlValue[] | null;
  host: GatewayLogsFilterUrlValue[] | null;
  requestId: GatewayLogsFilterUrlValue[] | null;
  startTime: number | null;
  endTime: number | null;
  since: string | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<GatewayLogsFilterOperator>([
  ...ALL_OPERATORS,
]);
