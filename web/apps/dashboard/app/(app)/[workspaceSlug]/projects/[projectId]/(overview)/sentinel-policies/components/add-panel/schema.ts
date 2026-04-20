import {
  type FirewallPolicy,
  type KeyauthPolicy,
  type MatchExpr,
  type RateLimitIdentifier,
  type RatelimitPolicy,
  SENTINEL_LIMITS,
  type StringMatch,
  firewallActionSchema,
  matchExprSchema,
  stringMatchModeSchema,
} from "@/lib/collections/deploy/sentinel-policies.schema";
import { P, match } from "@unkey/match";
import { z } from "zod";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
export type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";

const pathConditionSchema = z.object({
  id: z.string(),
  type: z.literal("path"),
  mode: stringMatchModeSchema,
  // Canonical stringMatchValue is min(1) — enforce here so users see a
  // field-level error instead of a generic 500 from savePolicies.
  value: z.string().min(1, "Value is required"),
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

// Header/queryParam conditions match against a stringMatch whose value must be
// non-empty (canonical stringMatchValue = min(1)) unless `present` is set.
// Refine on the union so the error attaches to the `value` field — users see a
// field-level error instead of a generic 500 from savePolicies.
export const matchConditionSchema = z
  .discriminatedUnion("type", [
    pathConditionSchema,
    methodConditionSchema,
    headerConditionSchema,
    queryParamConditionSchema,
  ])
  .superRefine((c, ctx) => {
    if (
      (c.type === "header" || c.type === "queryParam") &&
      !c.present &&
      (c.value ?? "").length === 0
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Value is required",
        path: ["value"],
      });
    }
  });

export type MatchConditionFormValues = z.infer<typeof matchConditionSchema>;

export const keyLocationTypeSchema = z.enum(["bearer", "header", "queryParam"]);
export type KeyLocationType = z.infer<typeof keyLocationTypeSchema>;

const keyLocationFormSchema = z
  .object({
    id: z.string(),
    locationType: keyLocationTypeSchema,
    name: z.string().optional(),
    stripPrefix: z.string().optional(),
  })
  .superRefine((loc, ctx) => {
    if (
      (loc.locationType === "header" || loc.locationType === "queryParam") &&
      (!loc.name || loc.name.length === 0)
    ) {
      ctx.addIssue({
        code: "custom",
        message:
          loc.locationType === "header" ? "Header name is required" : "Parameter name is required",
        path: ["name"],
      });
    }
  });

export type KeyLocationFormValues = z.infer<typeof keyLocationFormSchema>;

// ── Policy form schema ──────────────────────────────────────────────────
//
// Discriminated union on `type` so adding a new policy form (e.g. ratelimit)
// later is just one extra branch here + one branch in toSentinelPolicy below.
// `keyauth` and `firewall` are wired through today.

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

export const rateLimitIdentifierSourceSchema = z.enum([
  "remoteIp",
  "header",
  "authenticatedSubject",
  "path",
  "principalField",
]);
export type RateLimitIdentifierSource = z.infer<typeof rateLimitIdentifierSourceSchema>;

const ratelimitFormSchema = z
  .object({
    ...basePolicyFields,
    type: z.literal("ratelimit"),
    limit: z.number().int().min(1, "Limit must be at least 1"),
    windowMs: z.number().int().min(1, "Window must be at least 1ms"),
    identifierSource: rateLimitIdentifierSourceSchema,
    identifierValue: z.string(),
  })
  .superRefine((v, ctx) => {
    if (
      (v.identifierSource === "header" || v.identifierSource === "principalField") &&
      v.identifierValue.length === 0
    ) {
      ctx.addIssue({
        code: "custom",
        message:
          v.identifierSource === "header" ? "Header name is required" : "Field path is required",
        path: ["identifierValue"],
      });
    }
  });

// Firewall has a single action today (DENY) and no other configuration.
// The action is kept on the form so the wire payload stays self-describing
// and so adding more actions later is purely additive.
const firewallFormSchema = z.object({
  ...basePolicyFields,
  type: z.literal("firewall"),
  action: firewallActionSchema,
});

export const policyFormSchema = z.discriminatedUnion("type", [
  keyauthFormSchema,
  ratelimitFormSchema,
  firewallFormSchema,
]);
export type PolicyFormValues = z.infer<typeof policyFormSchema>;
export type PolicyType = PolicyFormValues["type"];

export const POLICY_TYPE_OPTIONS: { value: PolicyType; label: string }[] = [
  { value: "keyauth", label: "Key Auth" },
  { value: "ratelimit", label: "Rate Limit" },
  { value: "firewall", label: "Firewall" },
];

export function getDefaultCondition(
  type: MatchConditionFormValues["type"],
  id?: string,
): MatchConditionFormValues {
  const base = { id: id ?? crypto.randomUUID() };
  return match(type)
    .with("path", () => ({ ...base, type: "path" as const, mode: "exact" as const, value: "" }))
    .with("method", () => ({
      ...base,
      type: "method" as const,
      methods: [] as z.infer<typeof httpMethodSchema>[],
    }))
    .with("header", () => ({ ...base, type: "header" as const, name: "" }))
    .with("queryParam", () => ({ ...base, type: "queryParam" as const, name: "" }))
    .exhaustive();
}

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
      identifierSource: "remoteIp" as const,
      identifierValue: "",
    }))
    .with("firewall", () => ({
      ...base,
      type: "firewall" as const,
      action: "ACTION_DENY" as const,
    }))
    .exhaustive();
}

export function resolveTargetEnvs(
  selection: string,
  envASlug: string,
  envBSlug: string,
): { envA: boolean; envB: boolean } {
  return {
    envA: selection === "__all__" || selection === envASlug,
    envB: selection === "__all__" || selection === envBSlug,
  };
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

function toRateLimitIdentifier(
  source: RateLimitIdentifierSource,
  value: string,
): RateLimitIdentifier {
  return match(source)
    .returnType<RateLimitIdentifier>()
    .with("remoteIp", () => ({ remoteIp: {} }))
    .with("header", () => ({ header: { name: value } }))
    .with("authenticatedSubject", () => ({ authenticatedSubject: {} }))
    .with("path", () => ({ path: {} }))
    .with("principalField", () => ({ principalField: { path: value } }))
    .exhaustive();
}

export function toSentinelPolicy(
  values: PolicyFormValues,
  existingId?: string,
): KeyauthPolicy | RatelimitPolicy | FirewallPolicy {
  const id = existingId ?? crypto.randomUUID();
  const matchExprs = values.matchConditions.map(toMatchExpr);

  return match(values)
    .returnType<KeyauthPolicy | RatelimitPolicy | FirewallPolicy>()
    .with({ type: "keyauth" }, (v) => {
      const locations = v.locations.map((loc) =>
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
      );

      return {
        id,
        name: v.name,
        enabled: true,
        type: "keyauth" as const,
        keyauth: {
          keySpaceIds: v.keySpaceIds,
          locations,
          permissionQuery: v.permissionQuery,
        },
        match: matchExprs,
      };
    })
    .with({ type: "ratelimit" }, (v) => ({
      id,
      name: v.name,
      enabled: true,
      type: "ratelimit" as const,
      ratelimit: {
        limit: v.limit,
        windowMs: v.windowMs,
        identifier: toRateLimitIdentifier(v.identifierSource, v.identifierValue),
      },
      match: matchExprs,
    }))
    .with({ type: "firewall" }, (v) => ({
      id,
      name: v.name,
      enabled: true,
      type: "firewall" as const,
      firewall: { action: v.action },
      match: matchExprs,
    }))
    .exhaustive();
}

// ── Canonical → form conversion (inverse of toSentinelPolicy) ───────────

function stringMatchToMode(sm: StringMatch): { mode: "exact" | "prefix" | "regex"; value: string } {
  return match(sm)
    .returnType<{ mode: "exact" | "prefix" | "regex"; value: string }>()
    .with({ exact: P.string }, (s) => ({ mode: "exact", value: s.exact }))
    .with({ prefix: P.string }, (s) => ({ mode: "prefix", value: s.prefix }))
    .with({ regex: P.string }, (s) => ({ mode: "regex", value: s.regex }))
    .exhaustive();
}

// Match conditions and key locations have no id on the wire (they're protobuf
// oneofs), but the form needs a stable client-only id per row so React can key
// the list and the editor can address rows individually (update/delete a single
// condition without touching its siblings). We mint a fresh UUID on read here;
// it's discarded again on save by toSentinelPolicy.
function fromMatchExpr(raw: unknown): MatchConditionFormValues | null {
  const parsed = matchExprSchema.safeParse(raw);
  if (!parsed.success) {
    return null;
  }
  const expr = parsed.data;
  const id = crypto.randomUUID();
  return match(expr)
    .returnType<MatchConditionFormValues>()
    .with({ path: P._ }, (e) => {
      const { mode, value } = stringMatchToMode(e.path.path);
      return { id, type: "path" as const, mode, value };
    })
    .with({ method: P._ }, (e) => ({ id, type: "method" as const, methods: e.method.methods }))
    .with({ header: P._ }, (e) => {
      if ("present" in e.header) {
        return { id, type: "header" as const, name: e.header.name, present: true as const };
      }
      const { mode, value } = stringMatchToMode(e.header.value);
      return { id, type: "header" as const, name: e.header.name, mode, value };
    })
    .with({ queryParam: P._ }, (e) => {
      if ("present" in e.queryParam) {
        return { id, type: "queryParam" as const, name: e.queryParam.name, present: true as const };
      }
      const { mode, value } = stringMatchToMode(e.queryParam.value);
      return { id, type: "queryParam" as const, name: e.queryParam.name, mode, value };
    })
    .exhaustive();
}

function fromRateLimitIdentifier(key: RateLimitIdentifier): {
  identifierSource: RateLimitIdentifierSource;
  identifierValue: string;
} {
  return match(key)
    .returnType<{ identifierSource: RateLimitIdentifierSource; identifierValue: string }>()
    .with({ remoteIp: P._ }, () => ({ identifierSource: "remoteIp" as const, identifierValue: "" }))
    .with({ header: P._ }, (k) => ({
      identifierSource: "header" as const,
      identifierValue: k.header.name,
    }))
    .with({ authenticatedSubject: P._ }, () => ({
      identifierSource: "authenticatedSubject" as const,
      identifierValue: "",
    }))
    .with({ path: P._ }, () => ({ identifierSource: "path" as const, identifierValue: "" }))
    .with({ principalField: P._ }, (k) => ({
      identifierSource: "principalField" as const,
      identifierValue: k.principalField.path,
    }))
    .exhaustive();
}

export function fromSentinelPolicy(
  policy: SentinelPolicy,
  environmentId: string,
): PolicyFormValues {
  const matchConditions: MatchConditionFormValues[] = (policy.match ?? [])
    .map(fromMatchExpr)
    .filter((c): c is MatchConditionFormValues => c !== null);

  return match(policy)
    .with({ type: "keyauth" }, (p) => {
      const locations: KeyLocationFormValues[] = (p.keyauth.locations ?? []).map((loc) => {
        const id = crypto.randomUUID();
        return match(loc)
          .returnType<KeyLocationFormValues>()
          .with({ bearer: P._ }, () => ({ id, locationType: "bearer" as const }))
          .with({ header: P._ }, (l) => ({
            id,
            locationType: "header" as const,
            name: l.header.name,
            ...(l.header.stripPrefix ? { stripPrefix: l.header.stripPrefix } : {}),
          }))
          .with({ queryParam: P._ }, (l) => ({
            id,
            locationType: "queryParam" as const,
            name: l.queryParam.name,
          }))
          .exhaustive();
      });

      return {
        type: "keyauth" as const,
        name: p.name,
        environmentId,
        matchConditions,
        keySpaceIds: p.keyauth.keySpaceIds,
        locations,
        permissionQuery: p.keyauth.permissionQuery ?? "",
      };
    })
    .with({ type: "ratelimit" }, (p) => {
      const { identifierSource, identifierValue } = fromRateLimitIdentifier(p.ratelimit.identifier);
      return {
        type: "ratelimit" as const,
        name: p.name,
        environmentId,
        matchConditions,
        limit: p.ratelimit.limit,
        windowMs: p.ratelimit.windowMs,
        identifierSource,
        identifierValue,
      };
    })
    .with({ type: "firewall" }, (p) => {
      const matchConditions: MatchConditionFormValues[] = (p.match ?? [])
        .map(fromMatchExpr)
        .filter((c): c is MatchConditionFormValues => c !== null);

      return {
        type: "firewall" as const,
        name: p.name,
        environmentId,
        matchConditions,
        action: p.firewall.action,
      };
    })
    .exhaustive();
}
