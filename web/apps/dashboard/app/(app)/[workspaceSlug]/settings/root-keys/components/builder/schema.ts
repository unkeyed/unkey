import { z } from "zod";
import { CATALOGUES, catalogueRows } from "./lib/catalogue";
import { ACTIONS, ALL_INSTANCES, RESOURCE_SCOPES } from "./lib/catalogue.types";
import { type Policy, policyError } from "./lib/policy";

const instancesSchema = z.union([
  z.tuple([z.literal(ALL_INSTANCES)]),
  z.array(z.string().min(1)).nonempty("Select one or more instances."),
]);

export const policySchema = z
  .object({
    scope: z.enum(RESOURCE_SCOPES),
    instances: instancesSchema,
    selection: z.record(z.string(), z.array(z.enum(ACTIONS)).nonempty()),
  })
  .superRefine((policy, ctx) => {
    const rows = new Set(catalogueRows(CATALOGUES[policy.scope]).map((row) => row.id));
    for (const rowId of Object.keys(policy.selection)) {
      if (!rows.has(rowId)) {
        ctx.addIssue({
          code: "custom",
          message: `The ${policy.scope} scope has no "${rowId}" permission.`,
          path: ["selection", rowId],
        });
      }
    }
  }) satisfies z.ZodType<Policy>;

const completePolicySchema = policySchema.superRefine((policy, ctx) => {
  const message = policyError(policy);
  if (message !== null) {
    ctx.addIssue({ code: "custom", message });
  }
});

export const nameSchema = z.string().trim().min(1, "Give this key a name.");

export const rootKeySchema = z.object({
  name: nameSchema,
  policies: z.array(completePolicySchema).min(1, "Grant at least one permission."),
});

export const legacyRootKeySchema = z.object({
  name: nameSchema,
});

export type RootKeyFormValues = z.infer<typeof rootKeySchema>;
export type LegacyRootKeyFormValues = z.infer<typeof legacyRootKeySchema>;

export const rootKeyDefaultValues: RootKeyFormValues = {
  name: "",
  policies: [],
};
