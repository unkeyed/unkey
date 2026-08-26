import { z } from "zod";
import { CATALOGUES, catalogueRows } from "./lib/catalogue";
import { ACTIONS, ALL_INSTANCES, RESOURCE_SCOPES } from "./lib/catalogue.types";
import type { Policy } from "./lib/policy";

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

export const rootKeySchema = z.object({
  name: z.string().trim().min(1, "Give this key a name."),
  policies: z.array(policySchema),
});

export type RootKeyFormValues = z.infer<typeof rootKeySchema>;

export const rootKeyDefaultValues: RootKeyFormValues = {
  name: "",
  policies: [],
};
