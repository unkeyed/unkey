import {
  type KeyauthPolicy,
  type MatchExpr,
  SENTINEL_LIMITS,
  type StringMatch,
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

export function toSentinelPolicy(
  values: PolicyFormValues,
  existingId?: string,
): KeyauthPolicy {
  const id = existingId ?? crypto.randomUUID();
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

// ── Canonical → form conversion (inverse of toSentinelPolicy) ───────────

function stringMatchToMode(sm: {
  exact?: string;
  prefix?: string;
  regex?: string;
}): { mode: "exact" | "prefix" | "regex"; value: string } {
  if (typeof sm.exact === "string") return { mode: "exact", value: sm.exact };
  if (typeof sm.prefix === "string") return { mode: "prefix", value: sm.prefix };
  if (typeof sm.regex === "string") return { mode: "regex", value: sm.regex };
  return { mode: "exact", value: "" };
}

// Match conditions and key locations have no id on the wire (they're protobuf
// oneofs), but the form needs a stable client-only id per row so React can key
// the list and the editor can address rows individually (update/delete a single
// condition without touching its siblings). We mint a fresh UUID on read here;
// it's discarded again on save by toSentinelPolicy.
function fromMatchExpr(
  expr: Record<string, unknown>,
): MatchConditionFormValues {
  const id = crypto.randomUUID();
  if ("path" in expr) {
    const p = expr.path as { path: { exact?: string; prefix?: string; regex?: string } };
    const { mode, value } = stringMatchToMode(p.path);
    return { id, type: "path", mode, value };
  }
  if ("method" in expr) {
    const m = expr.method as { methods: z.infer<typeof httpMethodSchema>[] };
    return { id, type: "method", methods: m.methods };
  }
  if ("header" in expr) {
    const h = expr.header as {
      name: string;
      present?: true;
      value?: { exact?: string; prefix?: string; regex?: string };
    };
    if (h.present) {
      return { id, type: "header", name: h.name, present: true };
    }
    const { mode, value } = stringMatchToMode(h.value ?? {});
    return { id, type: "header", name: h.name, mode, value };
  }
  if ("queryParam" in expr) {
    const q = expr.queryParam as {
      name: string;
      present?: true;
      value?: { exact?: string; prefix?: string; regex?: string };
    };
    if (q.present) {
      return { id, type: "queryParam", name: q.name, present: true };
    }
    const { mode, value } = stringMatchToMode(q.value ?? {});
    return { id, type: "queryParam", name: q.name, mode, value };
  }
  throw new Error("unknown match expression");
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

      const matchConditions: MatchConditionFormValues[] = (p.match ?? []).map((expr) =>
        fromMatchExpr(expr as unknown as Record<string, unknown>),
      );

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
