import type {
  MatchExpr,
  SentinelPolicy,
  StringMatch,
} from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { match } from "@unkey/match";
import { z } from "zod";

// ── Match condition form schema ─────────────────────────────────────────

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

// ── Key location form schema ────────────────────────────────────────────

export const keyLocationTypeSchema = z.enum(["bearer", "header", "queryParam"]);
export type KeyLocationType = z.infer<typeof keyLocationTypeSchema>;

const keyLocationFormSchema = z.object({
  id: z.string(),
  locationType: keyLocationTypeSchema,
  name: z.string().optional(),
  stripPrefix: z.string().optional(),
});

export type KeyLocationFormValues = z.infer<typeof keyLocationFormSchema>;

// ── Rate limit key source ───────────────────────────────────────────────

export const rateLimitKeySourceSchema = z.enum([
  "remoteIp",
  "header",
  "authenticatedSubject",
  "path",
  "principalClaim",
]);
export type RateLimitKeySource = z.infer<typeof rateLimitKeySourceSchema>;

// ── Policy form schema (discriminated union on type) ────────────────────

const basePolicyFields = {
  name: z.string().min(1, "Name is required"),
  environmentId: z.string(),
  matchConditions: z.array(matchConditionSchema),
};

const keyauthSchema = z.object({
  ...basePolicyFields,
  type: z.literal("keyauth"),
  keySpaceIds: z.array(z.string()).min(1, "Select at least one keyspace"),
  locations: z.array(keyLocationFormSchema),
  permissionQuery: z.string(),
});

const ratelimitSchema = z.object({
  ...basePolicyFields,
  type: z.literal("ratelimit"),
  limit: z.number().min(1, "Limit must be at least 1"),
  windowMs: z.number().min(1, "Window must be at least 1ms"),
  keySource: rateLimitKeySourceSchema,
  keyValue: z.string(),
});

export const policyFormSchema = z.discriminatedUnion("type", [keyauthSchema, ratelimitSchema]);

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

  return match(type)
    .with("keyauth", () => ({
      ...base,
      type: "keyauth" as const,
      keySpaceIds: [],
      locations: [],
      permissionQuery: "",
    }))
    .with("ratelimit", () => ({
      ...base,
      type: "ratelimit" as const,
      limit: 100,
      windowMs: 60000,
      keySource: "remoteIp" as const,
      keyValue: "",
    }))
    .exhaustive();
}

// ── Form → protojson conversion ─────────────────────────────────────────

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

export function toSentinelPolicy(values: PolicyFormValues): SentinelPolicy {
  const id = crypto.randomUUID();
  const matchExprs =
    values.matchConditions.length > 0 ? values.matchConditions.map(toMatchExpr) : undefined;

  const base = { id, name: values.name, enabled: true, match: matchExprs };

  return match(values)
    .with({ type: "keyauth" }, (v) => {
      const locations =
        v.locations.length > 0
          ? v.locations.map((loc) =>
              match(loc.locationType)
                .with("bearer", () => ({ bearer: {} }))
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
        ...base,
        type: "keyauth" as const,
        keyauth: {
          keySpaceIds: v.keySpaceIds,
          ...(locations ? { locations } : {}),
          ...(v.permissionQuery ? { permissionQuery: v.permissionQuery } : {}),
        },
      };
    })
    .with({ type: "ratelimit" }, (v) => {
      const key = match(v.keySource)
        .with("remoteIp", () => ({ remoteIp: {} }))
        .with("header", () => ({ header: { name: v.keyValue } }))
        .with("authenticatedSubject", () => ({ authenticatedSubject: {} }))
        .with("path", () => ({ path: {} }))
        .with("principalClaim", () => ({ principalClaim: { claimName: v.keyValue } }))
        .exhaustive();

      return {
        ...base,
        type: "ratelimit" as const,
        ratelimit: { limit: v.limit, windowMs: v.windowMs, key },
      };
    })
    .exhaustive();
}
