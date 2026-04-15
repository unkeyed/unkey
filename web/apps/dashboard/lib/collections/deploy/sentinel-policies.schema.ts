/**
 * Canonical sentinel policy schemas — single source of truth.
 *
 * Wire shape vs. Go service:
 *   The Go sentinel service (svc/sentinel) parses these blobs as protojson with
 *   `DiscardUnknown: true`. Policy is a protobuf oneof keyed on `keyauth | firewall | ...`,
 *   with no `type` discriminator field. We keep a client-side `type` field to drive
 *   zod's discriminated union and the UI router; the Go side silently ignores it.
 *
 *   `keyauth` and `firewall` are supported today. Add a new branch by extending the
 *   discriminated union below and wiring it through `fromWirePolicy` — the
 *   collection's per-type routing will pick it up.
 */
import { z } from "zod";

// ── Limits ──────────────────────────────────────────────────────────────

export const SENTINEL_LIMITS = {
  maxPolicies: 10,
  maxKeyspacesPerPolicy: 5,
  maxMatchExprsPerPolicy: 10,
  // Documented in svc/sentinel/proto/policies/v1/keyauth.proto:60
  // ("Limits: maximum 1000 characters, maximum 100 permission terms").
  permissionQueryMaxLength: 1000,
} as const;

// ── String match (protojson oneof: exact | prefix | regex) ──────────────

export const stringMatchModeSchema = z.enum(["exact", "prefix", "regex"]);
export type StringMatchMode = z.infer<typeof stringMatchModeSchema>;

const stringMatchBase = { ignoreCase: z.boolean().optional() } as const;
const stringMatchValue = z.string().min(1);

export const stringMatchSchema = z.union([
  z.object({ ...stringMatchBase, exact: stringMatchValue }).strict(),
  z.object({ ...stringMatchBase, prefix: stringMatchValue }).strict(),
  z.object({ ...stringMatchBase, regex: stringMatchValue }).strict(),
]);
export type StringMatch = z.infer<typeof stringMatchSchema>;

// ── Match expressions (protojson oneof: path | method | header | queryParam) ─

const httpMethod = z.enum(["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]);

export const matchExprSchema = z.union([
  z.object({ path: z.object({ path: stringMatchSchema }).strict() }).strict(),
  z
    .object({
      method: z.object({ methods: z.array(httpMethod).min(1) }).strict(),
    })
    .strict(),
  z
    .object({
      header: z
        .object({ name: z.string().min(1) })
        .and(
          z.union([z.object({ present: z.literal(true) }), z.object({ value: stringMatchSchema })]),
        ),
    })
    .strict(),
  z
    .object({
      queryParam: z
        .object({ name: z.string().min(1) })
        .and(
          z.union([z.object({ present: z.literal(true) }), z.object({ value: stringMatchSchema })]),
        ),
    })
    .strict(),
]);
export type MatchExpr = z.infer<typeof matchExprSchema>;

// ── Key location (protojson oneof: bearer | header | queryParam) ────────

export const keyLocationSchema = z.union([
  z.object({ bearer: z.object({}).strict() }).strict(),
  z
    .object({
      header: z
        .object({
          name: z.string().min(1),
          stripPrefix: z.string().optional(),
        })
        .strict(),
    })
    .strict(),
  z
    .object({
      queryParam: z.object({ name: z.string().min(1) }).strict(),
    })
    .strict(),
]);
export type KeyLocation = z.infer<typeof keyLocationSchema>;

// ── Common policy fields ────────────────────────────────────────────────

const policyBase = {
  id: z.string().min(1),
  name: z.string().min(1),
  enabled: z.boolean(),
  match: z.array(matchExprSchema).max(SENTINEL_LIMITS.maxMatchExprsPerPolicy).optional(),
} as const;

// ── KeyAuth policy ──────────────────────────────────────────────────────

export const keyauthPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("keyauth"),
    keyauth: z
      .object({
        keySpaceIds: z.array(z.string().min(1)).min(1).max(SENTINEL_LIMITS.maxKeyspacesPerPolicy),
        locations: z.array(keyLocationSchema).optional(),
        permissionQuery: z.string().max(SENTINEL_LIMITS.permissionQueryMaxLength).optional(),
      })
      .strict(),
  })
  .strict();
export type KeyauthPolicy = z.infer<typeof keyauthPolicySchema>;

// ── RateLimit policy ───────────────────────────────────────────────────

const rateLimitKeySchema = z.union([
  z.object({ remoteIp: z.object({}).strict() }).strict(),
  z.object({ header: z.object({ name: z.string().min(1) }).strict() }).strict(),
  z.object({ authenticatedSubject: z.object({}).strict() }).strict(),
  z.object({ path: z.object({}).strict() }).strict(),
  z
    .object({
      principalField: z.object({ path: z.string().min(1) }).strict(),
    })
    .strict(),
]);
export type RateLimitKey = z.infer<typeof rateLimitKeySchema>;

export const ratelimitPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("ratelimit"),
    ratelimit: z
      .object({
        limit: z.number().int().min(1),
        windowMs: z.number().int().min(1),
        key: rateLimitKeySchema,
      })
      .strict(),
  })
  .strict();

// ── Firewall policy ─────────────────────────────────────────────────────

// Wire values match sentinel.v1.Action enum names. Kept as string literals so
// protojson round-trips them by name rather than numeric value. The MVP only
// has ACTION_DENY; the enum exists so additional outcomes can land later
// without changing the schema shape.
export const firewallActionSchema = z.enum(["ACTION_DENY"]);
export type FirewallAction = z.infer<typeof firewallActionSchema>;

export const firewallPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("firewall"),
    firewall: z
      .object({
        action: firewallActionSchema,
      })
      .strict(),
  })
  .strict();

export type RatelimitPolicy = z.infer<typeof ratelimitPolicySchema>;
export type FirewallPolicy = z.infer<typeof firewallPolicySchema>;

// ── Sentinel policy (discriminated union — extend with new types here) ──

export const sentinelPolicySchema = z.discriminatedUnion("type", [
  keyauthPolicySchema,
  ratelimitPolicySchema,
  firewallPolicySchema,
]);
export type SentinelPolicy = z.infer<typeof sentinelPolicySchema>;
export type SentinelPolicyType = SentinelPolicy["type"];

// ── Top-level config blob (what's stored in appRuntimeSettings.sentinelConfig) ──

export const sentinelConfigSchema = z
  .object({
    policies: z.array(sentinelPolicySchema).max(SENTINEL_LIMITS.maxPolicies),
  })
  .strict();
export type SentinelConfig = z.infer<typeof sentinelConfigSchema>;

/**
 * Strip the client-side `type` discriminator before serializing to the wire.
 * Go's protojson parser uses `DiscardUnknown: true`, so it would tolerate the
 * extra field — but we keep the on-disk blob clean so it round-trips through
 * any future stricter parser.
 */
export function toWirePolicy(p: SentinelPolicy): Record<string, unknown> {
  const { type: _type, ...rest } = p;
  return rest;
}

/**
 * Re-attach the client-side `type` discriminator when reading a blob back.
 * Looks at which oneof key is present and infers `type`.
 */
export function fromWirePolicy(raw: unknown): SentinelPolicy {
  if (typeof raw !== "object" || raw === null) {
    throw new Error("policy must be an object");
  }
  const obj: Record<string, unknown> = { ...(raw as Record<string, unknown>) };
  if ("keyauth" in obj) {
    return sentinelPolicySchema.parse({ ...obj, type: "keyauth" });
  }
  if ("ratelimit" in obj) {
    return sentinelPolicySchema.parse({ ...obj, type: "ratelimit" });
  }
  if ("firewall" in obj) {
    return sentinelPolicySchema.parse({ ...obj, type: "firewall" });
  }
  throw new Error("unknown sentinel policy variant");
}
