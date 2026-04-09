import {
  type KeyauthPolicy,
  type MatchExpr,
  SENTINEL_LIMITS,
  type StringMatch,
  matchExprSchema,
  stringMatchModeSchema,
} from "@/lib/collections/deploy/sentinel-policies.schema";
import { match } from "@unkey/match";
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

export function toSentinelPolicy(values: PolicyFormValues, existingId?: string): KeyauthPolicy {
  const id = existingId ?? crypto.randomUUID();
  const matchExprs = values.matchConditions.map(toMatchExpr);

  return match(values)
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
    .exhaustive();
}

// ── Canonical → form conversion (inverse of toSentinelPolicy) ───────────

function stringMatchToMode(sm: {
  exact?: string;
  prefix?: string;
  regex?: string;
}): { mode: "exact" | "prefix" | "regex"; value: string } {
  if (typeof sm.exact === "string") {
    return { mode: "exact", value: sm.exact };
  }
  if (typeof sm.prefix === "string") {
    return { mode: "prefix", value: sm.prefix };
  }
  if (typeof sm.regex === "string") {
    return { mode: "regex", value: sm.regex };
  }
  return { mode: "exact", value: "" };
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
  if ("path" in expr) {
    const { mode, value } = stringMatchToMode(expr.path.path);
    return { id, type: "path", mode, value };
  }
  if ("method" in expr) {
    return { id, type: "method", methods: expr.method.methods };
  }
  if ("header" in expr) {
    if ("present" in expr.header) {
      return { id, type: "header", name: expr.header.name, present: true };
    }
    const { mode, value } = stringMatchToMode(expr.header.value);
    return { id, type: "header", name: expr.header.name, mode, value };
  }
  if ("queryParam" in expr) {
    if ("present" in expr.queryParam) {
      return { id, type: "queryParam", name: expr.queryParam.name, present: true };
    }
    const { mode, value } = stringMatchToMode(expr.queryParam.value);
    return { id, type: "queryParam", name: expr.queryParam.name, mode, value };
  }
  return null;
}

export function fromSentinelPolicy(
  policy: SentinelPolicy,
  environmentId: string,
): PolicyFormValues {
  return match(policy)
    .with({ type: "keyauth" }, (p) => {
      const locations: KeyLocationFormValues[] = (p.keyauth.locations ?? []).map((loc) => {
        const id = crypto.randomUUID();
        if ("bearer" in loc) {
          return { id, locationType: "bearer" };
        }
        if ("header" in loc) {
          return {
            id,
            locationType: "header",
            name: loc.header.name,
            ...(loc.header.stripPrefix ? { stripPrefix: loc.header.stripPrefix } : {}),
          };
        }
        return { id, locationType: "queryParam", name: loc.queryParam.name };
      });

      const matchConditions: MatchConditionFormValues[] = (p.match ?? [])
        .map(fromMatchExpr)
        .filter((c): c is MatchConditionFormValues => c !== null);

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
    .exhaustive();
}
