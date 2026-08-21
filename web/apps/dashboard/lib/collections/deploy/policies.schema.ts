/**
 * Canonical gateway policy schemas — single source of truth.
 *
 * These mirror `openapi.Policy` / `PolicyResponse` field for field, with one
 * addition: `type`. The API discriminates variants by which of `keyauth`,
 * `ratelimit`, ... is set, which zod cannot switch on, so `fromWirePolicy`
 * attaches `type` on read. `type` and the server-owned `id` are dropped
 * again by the SDK's own outbound schema, which ignores unknown keys.
 *
 * Add a policy type by extending the union below and wiring it through
 * `fromWirePolicy`.
 */
import { z } from "zod";

// ── Limits ──────────────────────────────────────────────────────────────

// Pre-flight only: the generated SDK carries none of the spec's size bounds, so
// these drive inline form errors and "N / max" counters instead of a server 400.
// Keep each in step with its counterpart under svc/api/openapi/spec. Too tight
// and a stored policy the API accepts fails to open for editing; too loose and
// the save 400s anyway.
export const POLICY_LIMITS = {
  maxPolicies: 50,
  maxNameLength: 256,
  maxKeyspacesPerPolicy: 5,
  maxMatchExprsPerPolicy: 10,
  permissionQueryMaxLength: 1000,
  maxRatelimitsPerKeyauth: 10,
  maxIdentifiersPerRatelimit: 5,
} as const;

// protojson emits int64 fields as JSON strings (proto3 JSON mapping), while
// the dashboard writes plain numbers. Accept both, normalize to number.
const wireInt64 = z
  .union([z.number(), z.string().regex(/^\d+$/).transform(Number)])
  .pipe(z.number().int().min(1));

// Like wireInt64 but allows 0, for fields where zero is meaningful (e.g. a
// keyauth credits override of 0 verifies the key without spending credits).
const wireInt64NonNegative = z
  .union([z.number(), z.string().regex(/^\d+$/).transform(Number)])
  .pipe(z.number().int().min(0));

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
  match: z.array(matchExprSchema).max(POLICY_LIMITS.maxMatchExprsPerPolicy).optional(),
} as const;

// ── KeyAuth policy ──────────────────────────────────────────────────────

// Mirrors frontline.v1.KeyRatelimit. `name` references a rate limit configured
// on the key (or its identity). `limit` + `duration` (ms) together define an
// inline override that need not exist on the key; the Go service only honors an
// override when BOTH are present, so we enforce both-or-neither here to avoid a
// silently-ignored partial override. `cost` defaults to 1 on the wire.
export const keyauthRatelimitSchema = z
  .object({
    name: z.string().min(1),
    limit: wireInt64.optional(),
    duration: wireInt64.optional(),
    cost: wireInt64.optional(),
  })
  .strict()
  .refine((r) => (r.limit === undefined) === (r.duration === undefined), {
    message: "Limit and duration must be set together",
    path: ["limit"],
  });
export type KeyauthRatelimit = z.infer<typeof keyauthRatelimitSchema>;

export const keyauthPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("keyauth"),
    keyauth: z
      .object({
        keyspaces: z.array(z.string().min(1)).min(1).max(POLICY_LIMITS.maxKeyspacesPerPolicy),
        locations: z.array(keyLocationSchema).optional(),
        permissionQuery: z.string().max(POLICY_LIMITS.permissionQueryMaxLength).optional(),
        ratelimits: z
          .array(keyauthRatelimitSchema)
          .max(POLICY_LIMITS.maxRatelimitsPerKeyauth)
          .optional(),
        // Usage credits deducted per matching request. Defaults to 1 on the
        // wire; 0 verifies the key without spending credits.
        credits: wireInt64NonNegative.optional(),
      })
      .strict(),
  })
  .strict();
export type KeyauthPolicy = z.infer<typeof keyauthPolicySchema>;

// ── RateLimit policy ───────────────────────────────────────────────────

const rateLimitIdentifierSchema = z.union([
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
export type RateLimitIdentifier = z.infer<typeof rateLimitIdentifierSchema>;

// A ratelimit carries the `identifiers` list (1-5 dimensions combined into
// one bucket key). The deprecated single `identifier` is still readable for
// old stored blobs; the Go side enforces exactly-one at write time and the
// refine mirrors it so locally constructed blobs fail fast instead of on
// save. New writes always use `identifiers`.
export const ratelimitPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("ratelimit"),
    ratelimit: z
      .object({
        limit: wireInt64,
        windowMs: wireInt64,
        identifier: rateLimitIdentifierSchema.optional(),
        identifiers: z
          .array(rateLimitIdentifierSchema)
          .min(1)
          .max(POLICY_LIMITS.maxIdentifiersPerRatelimit)
          .optional(),
      })
      .strict()
      .refine((r) => (r.identifier === undefined) !== (r.identifiers === undefined), {
        message: "Exactly one of identifier or identifiers must be set",
      }),
  })
  .strict();

// ── Firewall policy ─────────────────────────────────────────────────────

// Wire values match frontline.v1.Action enum names. Kept as string literals so
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

// ── OpenAPI Validation policy ──────────────────────────────────────────

export const openapiPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("openapi"),
    openapi: z.object({}).strict(),
  })
  .strict();
export type OpenapiPolicy = z.infer<typeof openapiPolicySchema>;

// ── Logging policy ──────────────────────────────────────────────────────

// Opts matching requests into capturing more request data in the request
// log (Requests tab). The gateway always records a base log entry per
// request (method, host, path, status, latency) — that cannot be turned
// off. This policy adds five independent captures on top: request headers
// (with user agent and client IP), response headers, request body,
// response body, and query data (query string and parameters). An empty
// `match` list matches every request, so a policy without match conditions
// captures the extras for all traffic.
// All fields are optional because protojson omits false booleans.
export const loggingPolicySchema = z
  .object({
    ...policyBase,
    type: z.literal("logging"),
    logging: z
      .object({
        requestHeaders: z.boolean().optional(),
        responseHeaders: z.boolean().optional(),
        requestBody: z.boolean().optional(),
        responseBody: z.boolean().optional(),
        query: z.boolean().optional(),
      })
      .strict(),
  })
  .strict();
export type LoggingPolicy = z.infer<typeof loggingPolicySchema>;

// ── Gateway policy (discriminated union — extend with new types here) ──

export const policySchema = z.discriminatedUnion("type", [
  keyauthPolicySchema,
  ratelimitPolicySchema,
  firewallPolicySchema,
  openapiPolicySchema,
  loggingPolicySchema,
]);
export type Policy = z.infer<typeof policySchema>;
export type PolicyType = Policy["type"];

/** A trailing space is invisible once rendered, so it cannot make a second policy. */
export function normalizePolicyName(name: string): string {
  return name.trim();
}

/**
 * A policy's identity. Type belongs in it because a name can repeat across
 * types: a firewall and a ratelimit policy both called "Guard" are two
 * policies, not one.
 */
export function policyIdentity(type: PolicyType, name: string): string {
  return `${type}:${normalizePolicyName(name)}`;
}

/** Attaches the `type` discriminator the API leaves implicit. */
export function fromWirePolicy(raw: unknown): Policy {
  if (typeof raw !== "object" || raw === null) {
    throw new Error("policy must be an object");
  }
  const obj: Record<string, unknown> = { ...(raw as Record<string, unknown>) };
  if ("keyauth" in obj) {
    return policySchema.parse({ ...obj, type: "keyauth" });
  }
  if ("ratelimit" in obj) {
    return policySchema.parse({ ...obj, type: "ratelimit" });
  }
  if ("firewall" in obj) {
    return policySchema.parse({ ...obj, type: "firewall" });
  }
  if ("openapi" in obj) {
    return policySchema.parse({ ...obj, type: "openapi" });
  }
  if ("logging" in obj) {
    return policySchema.parse({ ...obj, type: "logging" });
  }
  throw new Error("unknown gateway policy variant");
}
