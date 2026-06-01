import {
  type IdentitiesFilterOperator,
  identitiesFilterOperators,
} from "@/components/identities-table/schema/identities.schema";
import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { z } from "zod";

export type { IdentitiesFilterOperator };

export const identitiesFilterOperatorEnum = z.enum(identitiesFilterOperators);

export type FilterFieldConfigs = {
  externalId: StringConfig<IdentitiesFilterOperator>;
  lastUsedStart: NumberConfig<"is">;
  lastUsedEnd: NumberConfig<"is">;
  lastUsedSince: StringConfig<"is">;
};

export const identitiesFilterFieldConfig: FilterFieldConfigs = {
  externalId: {
    type: "string",
    operators: [...identitiesFilterOperators],
  },
  lastUsedStart: { type: "number", operators: ["is"] },
  lastUsedEnd: { type: "number", operators: ["is"] },
  lastUsedSince: { type: "string", operators: ["is"] },
};

// Only fields that appear in the filter popover UI
export const identitiesListFilterFieldNames = ["externalId"] as const;

export const identitiesFilterFieldEnum = z.enum([
  "externalId",
  "lastUsedStart",
  "lastUsedEnd",
  "lastUsedSince",
]);
export type IdentitiesFilterField = z.infer<typeof identitiesFilterFieldEnum>;

export type AllOperatorsUrlValue = {
  value: string;
  operator: IdentitiesFilterOperator;
};

export type IdentitiesFilterValue = FilterValue<IdentitiesFilterField, IdentitiesFilterOperator>;

export type IdentitiesQuerySearchParams = {
  externalId?: AllOperatorsUrlValue[] | null;
  lastUsedStart?: number | null;
  lastUsedEnd?: number | null;
  lastUsedSince?: string | null;
};

export const parseAsIdentitiesFilterArray = parseAsFilterValueArray<IdentitiesFilterOperator>([
  ...identitiesFilterOperators,
]);
