import { z } from "zod";
import { keysListFilterOperatorEnum } from "../../filters.schema";

const PAGINATION_LIMIT = 50;
const filterItemSchema = z.object({
  operator: keysListFilterOperatorEnum,
  value: z.string(),
});
const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const baseKeysSchema = z.object({
  keyAuthId: z.string(),
  names: baseFilterArraySchema,
  identities: baseFilterArraySchema,
  keyIds: baseFilterArraySchema,
  tags: baseFilterArraySchema,
});

export const keysQueryListPayload = baseKeysSchema.extend({
  cursor: z.string().nullish(),
  limit: z.number().default(PAGINATION_LIMIT),
});

export type KeysQueryListPayload = z.infer<typeof keysQueryListPayload>;
