import { z } from "zod";
import { ACTIONS, RESOURCE_SCOPES } from "./lib/catalogue.types";
import type { Policy } from "./lib/policy";

export const policySchema = z.object({
  scope: z.enum(RESOURCE_SCOPES),
  instances: z.array(z.string()),
  selection: z.record(z.string(), z.array(z.enum(ACTIONS))),
}) satisfies z.ZodType<Policy>;

export const rootKeySchema = z.object({
  name: z.string().trim().min(1, "Give this key a name."),
  policies: z.array(policySchema),
});

export type RootKeyFormValues = z.infer<typeof rootKeySchema>;

export const rootKeyDefaultValues: RootKeyFormValues = {
  name: "",
  policies: [],
};
