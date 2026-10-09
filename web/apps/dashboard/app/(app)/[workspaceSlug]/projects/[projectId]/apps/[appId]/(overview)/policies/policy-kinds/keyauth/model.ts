import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { plural } from "@/lib/fmt";
import { IconKey2Outline18 } from "@unkey/icons";
import { P, match } from "@unkey/match";
import { z } from "zod";
import { sharedFormFields } from "../shared";
import type { KindFields, PolicyKindModel } from "../types";

const keyLocationFormSchema = z.discriminatedUnion("locationType", [
  z.object({ locationType: z.literal("bearer") }),
  z.object({
    locationType: z.literal("header"),
    name: z.string().min(1, "Header name is required"),
    stripPrefix: z.string().optional(),
  }),
  z.object({
    locationType: z.literal("queryParam"),
    name: z.string().min(1, "Parameter name is required"),
  }),
]);

export type KeyLocationFormValues = z.infer<typeof keyLocationFormSchema>;
export type KeyLocationType = KeyLocationFormValues["locationType"];
export type KeyLocationOf<T extends KeyLocationType> = Extract<
  KeyLocationFormValues,
  { locationType: T }
>;

export const KEY_LOCATION_TYPES: readonly KeyLocationType[] = keyLocationFormSchema.options.map(
  (option) => option.shape.locationType.value,
);

export const keyLocationDefaults: { [T in KeyLocationType]: KeyLocationOf<T> } = {
  bearer: { locationType: "bearer" },
  header: { locationType: "header", name: "" },
  queryParam: { locationType: "queryParam", name: "" },
};

const ratelimitName = z.string().min(1, "Name is required");

// Mirrors keyauthRatelimitSchema. With override off the row references a
// named limit on the key. With it on, the user either overrides the cost
// alone, or defines an inline limit + duration (optionally with a cost).
// limit and duration must be set together: the Go service silently ignores a
// partial inline override, so we reject it here rather than letting it no-op.
const keyauthRatelimitFormSchema = z.discriminatedUnion("override", [
  z.object({ name: ratelimitName, override: z.literal(false) }),
  z
    .object({
      name: ratelimitName,
      override: z.literal(true),
      limit: z.number().int().min(1, "Limit must be at least 1").optional(),
      duration: z.number().int().min(1, "Duration must be at least 1ms").optional(),
      cost: z.number().int().min(1, "Cost must be at least 1").optional(),
    })
    .superRefine((r, ctx) => {
      const hasLimit = r.limit !== undefined;
      const hasDuration = r.duration !== undefined;
      if (hasLimit && !hasDuration) {
        ctx.addIssue({ code: "custom", message: "Duration is required", path: ["duration"] });
      }
      if (hasDuration && !hasLimit) {
        ctx.addIssue({ code: "custom", message: "Limit is required", path: ["limit"] });
      }
      if (!hasLimit && !hasDuration && r.cost === undefined) {
        ctx.addIssue({
          code: "custom",
          message: "Enter a cost, or a limit and duration, to override",
          path: ["cost"],
        });
      }
    }),
]);

export type KeyauthRatelimitFormValues = z.infer<typeof keyauthRatelimitFormSchema>;

export const keyauthFormSchema = z.object({
  ...sharedFormFields,
  type: z.literal("keyauth"),
  keyspaceIds: z
    .array(z.string())
    .min(1, "Select at least one keyspace")
    .max(POLICY_LIMITS.maxKeyspacesPerPolicy),
  locations: z.array(keyLocationFormSchema),
  permissionQuery: z.string().max(POLICY_LIMITS.permissionQueryMaxLength),
  ratelimits: z.array(keyauthRatelimitFormSchema).max(POLICY_LIMITS.maxRatelimitsPerKeyauth),
  // Undefined leaves the wire field unset, which the gateway treats as the
  // default cost of 1; 0 verifies the key without spending credits.
  credits: z.number().int().min(0, "Credits cannot be negative").optional(),
});

export type KeyauthFormValues = z.infer<typeof keyauthFormSchema>;

export const keyauth: PolicyKindModel<"keyauth", KeyauthFormValues> = {
  label: "Key Auth",
  Icon: IconKey2Outline18,
  does: "Checks the API key against your keyspaces. A valid key sets the identity. Only the first matching Key Auth runs.",
  rejects: [
    { status: 401, reason: "No key, or the key is not valid" },
    { status: 403, reason: "The key lacks the required permissions" },
    { status: 429, reason: "The key's rate limit or credits ran out" },
  ],
  defaults: () => ({
    type: "keyauth",
    keyspaceIds: [],
    locations: [],
    permissionQuery: "",
    ratelimits: [],
    credits: undefined,
  }),
  toWire: (v) => {
    const ratelimits = v.ratelimits.map((r) =>
      r.override
        ? {
            name: r.name,
            ...(r.limit !== undefined ? { limit: r.limit } : {}),
            ...(r.duration !== undefined ? { duration: r.duration } : {}),
            ...(r.cost !== undefined ? { cost: r.cost } : {}),
          }
        : { name: r.name },
    );
    return {
      type: "keyauth",
      keyauth: {
        keyspaces: v.keyspaceIds,
        locations: v.locations.map((loc) =>
          match(loc)
            .with({ locationType: "bearer" }, () => ({ bearer: {} }))
            .with({ locationType: "header" }, (l) => ({
              header: { name: l.name, ...(l.stripPrefix ? { stripPrefix: l.stripPrefix } : {}) },
            }))
            .with({ locationType: "queryParam" }, (l) => ({ queryParam: { name: l.name } }))
            .exhaustive(),
        ),
        permissionQuery: v.permissionQuery,
        ...(ratelimits.length > 0 ? { ratelimits } : {}),
        ...(v.credits !== undefined ? { credits: v.credits } : {}),
      },
    };
  },
  fromWire: (p) => ({
    type: "keyauth",
    keyspaceIds: p.keyauth.keyspaces,
    locations: (p.keyauth.locations ?? []).map((loc) =>
      match(loc)
        .returnType<KeyLocationFormValues>()
        .with({ bearer: P._ }, () => ({ locationType: "bearer" }))
        .with({ header: P._ }, (l) => ({
          locationType: "header",
          name: l.header.name,
          ...(l.header.stripPrefix ? { stripPrefix: l.header.stripPrefix } : {}),
        }))
        .with({ queryParam: P._ }, (l) => ({ locationType: "queryParam", name: l.queryParam.name }))
        .exhaustive(),
    ),
    permissionQuery: p.keyauth.permissionQuery ?? "",
    ratelimits: (p.keyauth.ratelimits ?? []).map((r) =>
      r.limit !== undefined || r.duration !== undefined || r.cost !== undefined
        ? { name: r.name, override: true, limit: r.limit, duration: r.duration, cost: r.cost }
        : { name: r.name, override: false },
    ),
    credits: p.keyauth.credits,
  }),
  summary: (v) => keyauthSummary(v, {}),
};

/** The form holds keyspace ids; the panel looks their names up and passes them. */
export function keyauthSummary(
  v: KindFields<KeyauthFormValues>,
  keyspaceNames: Readonly<Record<string, string>>,
): string {
  const keyspaces = v.keyspaceIds.map((id) => keyspaceNames[id] ?? id).join(", ");
  const extraLocations = v.locations.length - 1;
  const parts = [
    keyspaces || "No keyspace",
    extraLocations > 0
      ? `${locationText(v.locations[0])} +${extraLocations} more`
      : locationText(v.locations[0]),
  ];
  if (v.credits !== undefined) {
    parts.push(plural(v.credits, "credit"));
  }
  if (v.ratelimits.length > 0) {
    parts.push(plural(v.ratelimits.length, "rate limit"));
  }
  if (v.permissionQuery.trim()) {
    parts.push("permissions");
  }
  return parts.join(" · ");
}

function locationText(location: KeyLocationFormValues | undefined): string {
  if (!location) {
    return "Bearer token";
  }
  return match(location)
    .with({ locationType: "bearer" }, () => "Bearer token")
    .with({ locationType: "header" }, (l) => `header ${l.name || "…"}`)
    .with({ locationType: "queryParam" }, (l) => `query ?${l.name || "…"}`)
    .exhaustive();
}
