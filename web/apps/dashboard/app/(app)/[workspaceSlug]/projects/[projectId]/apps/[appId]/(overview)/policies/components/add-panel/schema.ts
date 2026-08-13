import {
  type FirewallPolicy,
  type KeyauthPolicy,
  type MatchExpr,
  type OpenapiPolicy,
  POLICY_LIMITS,
  type RateLimitIdentifier,
  type RatelimitPolicy,
  type StringMatch,
  firewallActionSchema,
  matchExprSchema,
  stringMatchModeSchema,
} from "@/lib/collections/deploy/policies.schema";
import { newUid } from "@unkey/id";
import { P, match } from "@unkey/match";
import { z } from "zod";

import type { Policy } from "@/lib/collections/deploy/policies.schema";
export type { Policy } from "@/lib/collections/deploy/policies.schema";

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
// later is just one extra branch here + one branch in toPolicy below.
// `keyauth` and `firewall` are wired through today.

const basePolicyFields = {
  name: z.string().min(1, "Name is required"),
  environmentId: z.string(),
  matchConditions: z.array(matchConditionSchema),
};

// Mirrors keyauthRatelimitSchema with a client-only `id` for React keying and a
// client-only `override` toggle. Discriminated on `override` so the two states
// are structurally distinct: override off is a bare named reference, override on
// carries the optional inline fields. The common case is referencing a named
// limit on the key (override off). When override is on the user can either:
//   - override the cost alone (the named limit's window is kept, only cost changes), or
//   - define an inline limit + duration (optionally with a cost) that need not
//     exist on the key.
// limit and duration are an inline pair and must be set together: the Go service
// silently ignores a partial inline override (only limit, or only duration), so
// we reject it here rather than letting it no-op on the wire. The fields stay
// optional on the override-on branch because the form holds them undefined while
// the user is still typing; the superRefine enforces the valid combinations.
const keyauthRatelimitFormSchema = z
  .discriminatedUnion("override", [
    z.object({
      id: z.string(),
      name: z.string().min(1, "Name is required"),
      override: z.literal(false),
    }),
    z.object({
      id: z.string(),
      name: z.string().min(1, "Name is required"),
      override: z.literal(true),
      limit: z.number().int().min(1, "Limit must be at least 1").optional(),
      duration: z.number().int().min(1, "Duration must be at least 1ms").optional(),
      cost: z.number().int().min(1, "Cost must be at least 1").optional(),
    }),
  ])
  .superRefine((r, ctx) => {
    if (!r.override) {
      return;
    }
    const hasLimit = r.limit !== undefined;
    const hasDuration = r.duration !== undefined;
    if (hasLimit && !hasDuration) {
      ctx.addIssue({ code: "custom", message: "Duration is required", path: ["duration"] });
    }
    if (hasDuration && !hasLimit) {
      ctx.addIssue({ code: "custom", message: "Limit is required", path: ["limit"] });
    }
    // Override is on but nothing was entered — require at least one of the
    // valid overrides (cost, or the limit+duration pair) so the toggle isn't a
    // no-op that serializes to a bare named reference.
    if (!hasLimit && !hasDuration && r.cost === undefined) {
      ctx.addIssue({
        code: "custom",
        message: "Enter a cost, or a limit and duration, to override",
        path: ["cost"],
      });
    }
  });

export type KeyauthRatelimitFormValues = z.infer<typeof keyauthRatelimitFormSchema>;

const keyauthFormSchema = z.object({
  ...basePolicyFields,
  type: z.literal("keyauth"),
  keySpaceIds: z
    .array(z.string())
    .min(1, "Select at least one keyspace")
    .max(POLICY_LIMITS.maxKeyspacesPerPolicy),
  locations: z.array(keyLocationFormSchema),
  permissionQuery: z.string().max(POLICY_LIMITS.permissionQueryMaxLength),
  ratelimits: z.array(keyauthRatelimitFormSchema).max(POLICY_LIMITS.maxRatelimitsPerKeyauth),
  // Usage credits deducted per matching request. Undefined leaves the wire
  // field unset, which the gateway treats as the default cost of 1; 0 verifies
  // the key without spending credits.
  credits: z.number().int().min(0, "Credits cannot be negative").optional(),
});

export const rateLimitIdentifierSourceSchema = z.enum([
  "remoteIp",
  "header",
  "authenticatedSubject",
  "path",
  "principalField",
]);
export type RateLimitIdentifierSource = z.infer<typeof rateLimitIdentifierSourceSchema>;

// One row of the identifiers list: a source plus its value for the sources
// that need one (header name, principal field path). `id` is client-only for
// React keying, minted on read and discarded on save like match conditions.
const ratelimitIdentifierRowSchema = z
  .object({
    id: z.string(),
    source: rateLimitIdentifierSourceSchema,
    value: z.string(),
  })
  .superRefine((row, ctx) => {
    if ((row.source === "header" || row.source === "principalField") && row.value.length === 0) {
      ctx.addIssue({
        code: "custom",
        message: row.source === "header" ? "Header name is required" : "Field path is required",
        path: ["value"],
      });
    }
  });

export type RatelimitIdentifierRowValues = z.infer<typeof ratelimitIdentifierRowSchema>;

const ratelimitFormSchema = z.object({
  ...basePolicyFields,
  type: z.literal("ratelimit"),
  limit: z.number().int().min(1, "Limit must be at least 1"),
  windowMs: z.number().int().min(1, "Window must be at least 1ms"),
  // Rows form the identifiers list; 2+ rows form a compound key where each
  // unique combination of resolved values gets its own counter.
  identifiers: z
    .array(ratelimitIdentifierRowSchema)
    .min(1, "Add at least one identifier")
    .max(POLICY_LIMITS.maxIdentifiersPerRatelimit),
});

// Firewall has a single action today (DENY) and no other configuration.
// The action is kept on the form so the wire payload stays self-describing
// and so adding more actions later is purely additive.
const firewallFormSchema = z.object({
  ...basePolicyFields,
  type: z.literal("firewall"),
  action: firewallActionSchema,
});

const openapiFormSchema = z.object({
  ...basePolicyFields,
  type: z.literal("openapi"),
});

export const policyFormSchema = z.discriminatedUnion("type", [
  keyauthFormSchema,
  ratelimitFormSchema,
  firewallFormSchema,
  openapiFormSchema,
]);
export type PolicyFormValues = z.infer<typeof policyFormSchema>;
export type PolicyType = PolicyFormValues["type"];

export const POLICY_TYPE_OPTIONS: { value: PolicyType; label: string }[] = [
  { value: "keyauth", label: "Key Auth" },
  { value: "ratelimit", label: "Rate Limit" },
  { value: "firewall", label: "Firewall" },
  { value: "openapi", label: "OpenAPI Validation" },
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
      ratelimits: [],
      credits: undefined,
    }))
    .with("ratelimit", () => ({
      ...base,
      type: "ratelimit" as const,
      limit: 100,
      windowMs: 60000,
      identifiers: [
        { id: crypto.randomUUID(), source: "remoteIp" as const, value: "" },
      ] as RatelimitIdentifierRowValues[],
    }))
    .with("firewall", () => ({
      ...base,
      type: "firewall" as const,
      action: "ACTION_DENY" as const,
    }))
    .with("openapi", () => ({
      ...base,
      type: "openapi" as const,
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

export function toPolicy(
  values: PolicyFormValues,
  existingId?: string,
): KeyauthPolicy | RatelimitPolicy | FirewallPolicy | OpenapiPolicy {
  const id = existingId ?? newUid("policy");
  const matchExprs = values.matchConditions.map(toMatchExpr);

  return match(values)
    .returnType<KeyauthPolicy | RatelimitPolicy | FirewallPolicy | OpenapiPolicy>()
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

      const ratelimits = v.ratelimits.map((r) => ({
        name: r.name,
        ...(r.override && r.limit !== undefined ? { limit: r.limit } : {}),
        ...(r.override && r.duration !== undefined ? { duration: r.duration } : {}),
        ...(r.override && r.cost !== undefined ? { cost: r.cost } : {}),
      }));

      return {
        id,
        name: v.name,
        enabled: true,
        type: "keyauth" as const,
        keyauth: {
          keySpaceIds: v.keySpaceIds,
          locations,
          permissionQuery: v.permissionQuery,
          ...(ratelimits.length > 0 ? { ratelimits } : {}),
          ...(v.credits !== undefined ? { credits: v.credits } : {}),
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
        // Always serialize the identifiers array, even for one row, so all
        // writes converge on the target shape. Deserialization still reads
        // the deprecated single identifier from old stored policies.
        identifiers: v.identifiers.map((row) => toRateLimitIdentifier(row.source, row.value)),
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
    .with({ type: "openapi" }, (v) => ({
      id,
      name: v.name,
      enabled: true,
      type: "openapi" as const,
      openapi: {},
      match: matchExprs,
    }))
    .exhaustive();
}

// ── Canonical → form conversion (inverse of toPolicy) ───────────

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
// it's discarded again on save by toPolicy.
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

function fromRateLimitIdentifier(key: RateLimitIdentifier): RatelimitIdentifierRowValues {
  const id = crypto.randomUUID();
  return match(key)
    .returnType<RatelimitIdentifierRowValues>()
    .with({ remoteIp: P._ }, () => ({ id, source: "remoteIp" as const, value: "" }))
    .with({ header: P._ }, (k) => ({
      id,
      source: "header" as const,
      value: k.header.name,
    }))
    .with({ authenticatedSubject: P._ }, () => ({
      id,
      source: "authenticatedSubject" as const,
      value: "",
    }))
    .with({ path: P._ }, () => ({ id, source: "path" as const, value: "" }))
    .with({ principalField: P._ }, (k) => ({
      id,
      source: "principalField" as const,
      value: k.principalField.path,
    }))
    .exhaustive();
}

export function fromPolicy(policy: Policy, environmentId: string): PolicyFormValues {
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

      const ratelimits: KeyauthRatelimitFormValues[] = (p.keyauth.ratelimits ?? []).map((r) => ({
        id: crypto.randomUUID(),
        name: r.name,
        override: r.limit !== undefined || r.duration !== undefined || r.cost !== undefined,
        limit: r.limit,
        duration: r.duration,
        cost: r.cost,
      }));

      return {
        type: "keyauth" as const,
        name: p.name,
        environmentId,
        matchConditions,
        keySpaceIds: p.keyauth.keySpaceIds,
        locations,
        permissionQuery: p.keyauth.permissionQuery ?? "",
        ratelimits,
        credits: p.keyauth.credits,
      };
    })
    .with({ type: "ratelimit" }, (p) => {
      // Wire blobs carry either the single identifier or the compound list;
      // both open in the same rows array.
      const wireIdentifiers =
        p.ratelimit.identifiers ?? (p.ratelimit.identifier ? [p.ratelimit.identifier] : []);
      return {
        type: "ratelimit" as const,
        name: p.name,
        environmentId,
        matchConditions,
        limit: p.ratelimit.limit,
        windowMs: p.ratelimit.windowMs,
        identifiers: wireIdentifiers.map(fromRateLimitIdentifier),
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
    .with({ type: "openapi" }, (p) => ({
      type: "openapi" as const,
      name: p.name,
      environmentId,
      matchConditions,
    }))
    .exhaustive();
}
