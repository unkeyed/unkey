import {
  rootKeysFilterOperatorEnum,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { z } from "zod";

const filterItemSchema = z.object({
  operator: rootKeysFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

type FilterFieldName = (typeof rootKeysListFilterFieldNames)[number];

const filterFieldsSchema = rootKeysListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<FilterFieldName, typeof baseFilterArraySchema>,
);

const baseRootKeysSchema = z.object(filterFieldsSchema);

export const rootKeysQueryPayload = baseRootKeysSchema.extend({
  limit: z.number().min(20).optional(),
  cursor: z.number().nullish(),
});

export type RootKeysQueryPayload = z.infer<typeof rootKeysQueryPayload>;
