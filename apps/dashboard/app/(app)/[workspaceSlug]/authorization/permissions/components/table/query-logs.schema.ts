import { z } from "zod";
import {
  permissionsFilterOperatorEnum,
  permissionsListFilterFieldNames,
} from "../../filters.schema";
const filterItemSchema = z.object({
  operator: permissionsFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const filterFieldsSchema = permissionsListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<string, typeof baseFilterArraySchema>,
);

const basePermissionsSchema = z.object(filterFieldsSchema);

export const permissionsQueryPayload = basePermissionsSchema.extend({
  cursor: z.number().nullish(),
});

export type PermissionsQueryPayload = z.infer<typeof permissionsQueryPayload>;
