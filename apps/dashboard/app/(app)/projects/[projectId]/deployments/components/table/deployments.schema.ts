import { z } from "zod";
import {
  deploymentListFilterFieldNames,
  deploymentListFilterOperatorEnum,
} from "../../filters.schema";

const filterItemSchema = z.object({
  operator: deploymentListFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const filterFieldsSchema = deploymentListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<string, typeof baseFilterArraySchema>,
);

const baseRootKeysSchema = z.object(filterFieldsSchema);

export const deploymentsInputSchema = baseRootKeysSchema.extend({
  cursor: z.number().nullish(),
});

export type DeploymentsInputSchema = z.infer<typeof deploymentsInputSchema>;
