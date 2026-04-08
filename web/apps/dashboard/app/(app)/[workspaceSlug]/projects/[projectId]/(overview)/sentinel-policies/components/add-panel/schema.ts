import {
  type KeyauthPolicy,
  type MatchExpr,
  SENTINEL_LIMITS,
  type StringMatch,
  stringMatchModeSchema,
} from "@/lib/collections/deploy/sentinel-policies.schema";
import { match } from "@unkey/match";
import { z } from "zod";

export type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";

const pathConditionSchema = z.object({
  id: z.string(),
  type: z.literal("path"),
  mode: stringMatchModeSchema,
  value: z.string(),
});

const httpMethodSchema = z.enum(["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]);

const methodConditionSchema = z.object({
  id: z.string(),
  type: z.literal("method"),
  methods: z.array(httpMethodSchema).min(1, "Select at least one method"),
});

const headerConditionSchema = z.object({
  id: z.string(),
  type: z.literal("header"),
  name: z.string().min(1, "Header name is required"),
  present: z.boolean().optional(),
  mode: stringMatchModeSchema.optional(),
  value: z.string().optional(),
});

const queryParamConditionSchema = z.object({
  id: z.string(),
  type: z.literal("queryParam"),
  name: z.string().min(1, "Param name is required"),
  present: z.boolean().optional(),
  mode: stringMatchModeSchema.optional(),
  value: z.string().optional(),
});

export const matchConditionSchema = z.discriminatedUnion("type", [
  pathConditionSchema,
  methodConditionSchema,
  headerConditionSchema,
  queryParamConditionSchema,
]);

export type MatchConditionFormValues = z.infer<typeof matchConditionSchema>;


export const keyLocationTypeSchema = z.enum(["bearer", "header", "queryParam"]);
export type KeyLocationType = z.infer<typeof keyLocationTypeSchema>;

const keyLocationFormSchema = z.object({
  id: z.string(),
  locationType: keyLocationTypeSchema,
  name: z.string().optional(),
  stripPrefix: z.string().optional(),
});

export type KeyLocationFormValues = z.infer<typeof keyLocationFormSchema>;

// ── Policy form schema ──────────────────────────────────────────────────
//
// Discriminated union on `type` so adding a new policy form (e.g. ratelimit)
// later is just one extra branch here + one branch in toSentinelPolicy below.
// Today only `keyauth` is wired through.

const basePolicyFields = {
  name: z.string().min(1, "Name is required"),
  environmentId: z.string(),
  matchConditions: z.array(matchConditionSchema),
};

const keyauthFormSchema = z.object({
  ...basePolicyFields,
  type: z.literal("keyauth"),
  keySpaceIds: z
    .array(z.string())
    .min(1, "Select at least one keyspace")
    .max(SENTINEL_LIMITS.maxKeyspacesPerPolicy),
  locations: z.array(keyLocationFormSchema),
  permissionQuery: z.string().max(SENTINEL_LIMITS.permissionQueryMaxLength),
});

export const policyFormSchema = z.discriminatedUnion("type", [keyauthFormSchema]);
export type PolicyFormValues = z.infer<typeof policyFormSchema>;
export type PolicyType = PolicyFormValues["type"];

export const POLICY_TYPE_OPTIONS: { value: PolicyType; label: string }[] = [
  { value: "keyauth", label: "Key Auth" },
];

export function getDefaultValues(type: PolicyType): PolicyFormValues {
  const base = {
    name: "",
    environmentId: "__all__",
    matchConditions: [],
  };

  return match(type)
    .with("keyauth", () => ({
      ...base,
      type: "keyauth" as const,
      keySpaceIds: [],
      locations: [],
      permissionQuery: "",
    }))
    .exhaustive();
}

// ── Form → canonical (protojson) conversion ─────────────────────────────

function toStringMatch(
  mode: "exact" | "prefix" | "regex",
  value: string,
  ignoreCase?: boolean,
): StringMatch {
  const base = ignoreCase ? { ignoreCase: true } : {};
  return match(mode)
    .returnType<StringMatch>()
    .with("exact", () => ({ ...base, exact: value }))
    .with("prefix", () => ({ ...base, prefix: value }))
    .with("regex", () => ({ ...base, regex: value }))
    .exhaustive();
}

function toMatchExpr(condition: MatchConditionFormValues): MatchExpr {
  return match(condition)
    .returnType<MatchExpr>()
    .with({ type: "path" }, (c) => ({
      path: { path: toStringMatch(c.mode, c.value) },
    }))
    .with({ type: "method" }, (c) => ({
      method: { methods: c.methods },
    }))
    .with({ type: "header" }, (c) =>
      c.present
        ? { header: { name: c.name, present: true } }
        : {
          header: {
            name: c.name,
            value: toStringMatch(c.mode ?? "exact", c.value ?? ""),
          },
        },
    )
    .with({ type: "queryParam" }, (c) =>
      c.present
        ? { queryParam: { name: c.name, present: true } }
        : {
          queryParam: {
            name: c.name,
            value: toStringMatch(c.mode ?? "exact", c.value ?? ""),
          },
        },
    )
    .exhaustive();
}

export function toSentinelPolicy(values: PolicyFormValues): KeyauthPolicy {
  const id = crypto.randomUUID();
  const matchExprs =
    values.matchConditions.length > 0 ? values.matchConditions.map(toMatchExpr) : undefined;

  return match(values)
    .with({ type: "keyauth" }, (v) => {
      const locations =
        v.locations.length > 0
          ? v.locations.map((loc) =>
            match(loc.locationType)
              .with("bearer", () => ({ bearer: {} as Record<string, never> }))
              .with("header", () => ({
                header: {
                  name: loc.name ?? "",
                  ...(loc.stripPrefix ? { stripPrefix: loc.stripPrefix } : {}),
                },
              }))
              .with("queryParam", () => ({ queryParam: { name: loc.name ?? "" } }))
              .exhaustive(),
          )
          : undefined;

      return {
        id,
        name: v.name,
        enabled: true,
        type: "keyauth" as const,
        keyauth: {
          keySpaceIds: v.keySpaceIds,
          ...(locations ? { locations } : {}),
          ...(v.permissionQuery ? { permissionQuery: v.permissionQuery } : {}),
        },
        ...(matchExprs ? { match: matchExprs } : {}),
      };
    })
    .exhaustive();
}
