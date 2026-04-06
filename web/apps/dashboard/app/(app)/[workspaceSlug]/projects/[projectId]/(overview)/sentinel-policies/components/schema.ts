import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { z } from "zod";

const stringMatchMode = z.enum(["exact", "prefix", "regex"]);

const pathConditionSchema = z.object({
  id: z.string(),
  type: z.literal("path"),
  mode: stringMatchMode,
  value: z.string(),
});

const methodConditionSchema = z.object({
  id: z.string(),
  type: z.literal("method"),
  methods: z.array(z.string()).min(1, "Select at least one method"),
});

const headerConditionSchema = z.object({
  id: z.string(),
  type: z.literal("header"),
  name: z.string().min(1, "Header name is required"),
  present: z.boolean().optional(),
  mode: stringMatchMode.optional(),
  value: z.string().optional(),
});

const queryParamConditionSchema = z.object({
  id: z.string(),
  type: z.literal("queryParam"),
  name: z.string().min(1, "Param name is required"),
  present: z.boolean().optional(),
  mode: stringMatchMode.optional(),
  value: z.string().optional(),
});

export const matchConditionSchema = z.discriminatedUnion("type", [
  pathConditionSchema,
  methodConditionSchema,
  headerConditionSchema,
  queryParamConditionSchema,
]);

export type MatchConditionFormValues = z.infer<typeof matchConditionSchema>;

const basePolicyFields = {
  name: z.string().min(1, "Name is required"),
  environmentId: z.string(),
  matchConditions: z.array(matchConditionSchema),
};

const keyauthSchema = z.object({
  ...basePolicyFields,
  type: z.literal("keyauth"),
  keySpaceIds: z.array(z.string()).min(1, "Select at least one keyspace"),
});

const ratelimitSchema = z.object({
  ...basePolicyFields,
  type: z.literal("ratelimit"),
  limit: z.number().min(1, "Limit must be at least 1"),
  windowMs: z.number().min(1, "Window must be at least 1ms"),
});

export const policyFormSchema = z.discriminatedUnion("type", [
  keyauthSchema,
  ratelimitSchema,
]);

export type PolicyFormValues = z.infer<typeof policyFormSchema>;

export type PolicyType = PolicyFormValues["type"];

export const POLICY_TYPE_OPTIONS: { value: PolicyType; label: string }[] = [
  { value: "keyauth", label: "Key Auth" },
  { value: "ratelimit", label: "Rate Limit" },
];

export function getDefaultValues(type: PolicyType): PolicyFormValues {
  const base = {
    name: "",
    environmentId: "__all__",
    matchConditions: [],
  };

  switch (type) {
    case "keyauth":
      return { ...base, type: "keyauth", keySpaceIds: [] };
    case "ratelimit":
      return { ...base, type: "ratelimit", limit: 100, windowMs: 60000 };
  }
}

export function toSentinelPolicy(values: PolicyFormValues): SentinelPolicy {
  const id = crypto.randomUUID();
  const matchPatch =
    values.matchConditions.length > 0 ? { match: { conditions: values.matchConditions } } : {};

  const base = { id, name: values.name, enabled: true, type: values.type, ...matchPatch };

  switch (values.type) {
    case "keyauth":
      return { ...base, type: "keyauth", keyauth: { keySpaceIds: values.keySpaceIds } };
    case "ratelimit":
      return {
        ...base,
        type: "ratelimit",
        ratelimit: { limit: values.limit, windowMs: values.windowMs },
      };
  }
}
