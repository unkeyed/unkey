import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { z } from "zod";
import { matchConditionSchema } from "./conditions/schema";

export const sharedFormFields = {
  name: z
    .string()
    .trim()
    .min(1, "Name is required")
    .max(
      POLICY_LIMITS.maxNameLength,
      `Name must be at most ${POLICY_LIMITS.maxNameLength} characters`,
    ),
  matchConditions: z.array(matchConditionSchema),
};

export type SharedFormField = keyof typeof sharedFormFields;
