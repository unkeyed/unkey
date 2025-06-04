import { z } from "zod";
import { rolesFilterOperatorEnum } from "../../filters.schema";

const filterItemSchema = z.object({
  operator: rolesFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const baseRolesSchema = z.object({
  description: baseFilterArraySchema,
  name: baseFilterArraySchema,
});

export const rolesQueryPayload = baseRolesSchema.extend({
  cursor: z.number().nullish(),
});

export type RolesQueryPayload = z.infer<typeof rolesQueryPayload>;
