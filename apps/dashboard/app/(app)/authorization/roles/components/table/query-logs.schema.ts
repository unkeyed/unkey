import { z } from "zod";
import { rolesFilterOperatorEnum, rolesListFilterFieldNames } from "../../filters.schema";

const filterItemSchema = z.object({
  operator: rolesFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const filterFieldsSchema = rolesListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<string, typeof baseFilterArraySchema>,
);

const baseRolesSchema = z.object(filterFieldsSchema);

export const rolesQueryPayload = baseRolesSchema.extend({
  cursor: z.number().nullish(),
});

export type RolesQueryPayload = z.infer<typeof rolesQueryPayload>;
