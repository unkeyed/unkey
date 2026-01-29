import { z } from "zod";
import { deploymentListFilterOperatorEnum } from "../../filters.schema";

const filterItemSchema = z.object({
  operator: deploymentListFilterOperatorEnum,
  value: z.string(),
});

export const deploymentListInputSchema = z.object({
  status: z.array(filterItemSchema).nullish(),
  environment: z.array(filterItemSchema).nullish(),
  branch: z.array(filterItemSchema).nullish(),
  startTime: z.number().nullish(),
  endTime: z.number().nullish(),
  since: z.string().nullish(),
  cursor: z.number().nullish(),
});

export type DeploymentListInputSchema = z.infer<typeof deploymentListInputSchema>;
