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

export function splitIpAllowlist(value: string): string[] {
  return value
    .split("\n")
    .map((entry) => entry.trim())
    .filter(Boolean);
}

export function changedIpAllowlist(
  value: string,
  previous: readonly string[],
): string[] | undefined {
  const next = splitIpAllowlist(value);
  if (next.length === previous.length && next.every((entry, index) => entry === previous[index])) {
    return undefined;
  }
  return next;
}

const ipEntrySchema = z.union([z.ipv4(), z.ipv6(), z.cidrv4(), z.cidrv6()]);
const ipAllowlistSchema = z.string().superRefine((value, ctx) => {
  const entries = splitIpAllowlist(value);
  if (entries.length > 100) {
    ctx.addIssue({ code: "custom", message: "Enter at most 100 IP addresses or CIDR ranges." });
    return;
  }
  const invalid = entries.find((entry) => !ipEntrySchema.safeParse(entry).success);
  if (invalid) {
    ctx.addIssue({
      code: "custom",
      message: `“${invalid}” is not a valid IP address or CIDR range.`,
    });
  }
});

export const rootKeySchema = z.object({
  name: nameSchema,
  ipAllowlist: ipAllowlistSchema,
  policies: z.array(completePolicySchema).min(1, "Grant at least one permission."),
});

export type RootKeyFormValues = z.infer<typeof rootKeySchema>;

export const rootKeyDefaultValues: RootKeyFormValues = {
  name: "",
  ipAllowlist: "",
  policies: [],
};
