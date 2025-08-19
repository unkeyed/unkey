import { z } from "zod";
import { rootKeysFilterOperatorEnum, rootKeysListFilterFieldNames } from "../../filters.schema";

const filterItemSchema = z.object({
  operator: rootKeysFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const filterFieldsSchema = rootKeysListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<string, typeof baseFilterArraySchema>,
);

const baseRootKeysSchema = z.object(filterFieldsSchema);

export const rootKeysQueryPayload = baseRootKeysSchema.extend({
  cursor: z.number().nullish(),
});

export type RootKeysQueryPayload = z.infer<typeof rootKeysQueryPayload>;
