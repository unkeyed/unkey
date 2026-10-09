import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { z } from "zod";
import { matchConditionSchema } from "./conditions/schema";

export const sharedFormFields = {
  // The name is the policy's identity, so trim before the length checks: a
  // spaces-only name can never pair, and a trailing space makes a second row
  // that renders the same as an existing one.
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
