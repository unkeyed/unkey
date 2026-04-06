import type {
  MatchExpr,
  SentinelPolicy,
  StringMatch,
} from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
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

  switch (type) {
    case "keyauth":
      return { ...base, type: "keyauth", keySpaceIds: [], locations: [], permissionQuery: "" };
    case "ratelimit":
      return {
        ...base,
        type: "ratelimit",
        limit: 100,
        windowMs: 60000,
        keySource: "remoteIp",
        keyValue: "",
      };
  }
}

// ── Form → protojson conversion ─────────────────────────────────────────

function toStringMatch(
  mode: "exact" | "prefix" | "regex",
  value: string,
  ignoreCase?: boolean,
): StringMatch {
  const base = ignoreCase ? { ignoreCase: true } : {};
  switch (mode) {
    case "exact":
      return { ...base, exact: value } as StringMatch;
    case "prefix":
      return { ...base, prefix: value } as StringMatch;
    case "regex":
      return { ...base, regex: value } as StringMatch;
  }
}

function toMatchExpr(condition: MatchConditionFormValues): MatchExpr {
  switch (condition.type) {
    case "path":
      return { path: { path: toStringMatch(condition.mode, condition.value) } };
    case "method":
      return { method: { methods: condition.methods } };
    case "header":
      return condition.present
        ? { header: { name: condition.name, present: true } }
        : {
            header: {
              name: condition.name,
              value: toStringMatch(condition.mode ?? "exact", condition.value ?? ""),
            },
          };
    case "queryParam":
      return condition.present
        ? { queryParam: { name: condition.name, present: true } }
        : {
            queryParam: {
              name: condition.name,
              value: toStringMatch(condition.mode ?? "exact", condition.value ?? ""),
            },
          };
  }
}

export function toSentinelPolicy(values: PolicyFormValues): SentinelPolicy {
  const id = crypto.randomUUID();
  const match =
    values.matchConditions.length > 0 ? values.matchConditions.map(toMatchExpr) : undefined;

  const base = { id, name: values.name, enabled: true, type: values.type, match };

  switch (values.type) {
    case "keyauth": {
      const locations =
        values.locations.length > 0
          ? values.locations.map((loc) => {
              switch (loc.locationType) {
                case "bearer":
                  return { bearer: {} };
                case "header":
                  return {
                    header: {
                      name: loc.name ?? "",
                      ...(loc.stripPrefix ? { stripPrefix: loc.stripPrefix } : {}),
                    },
                  };
                case "queryParam":
                  return { queryParam: { name: loc.name ?? "" } };
              }
            })
          : undefined;

      return {
        ...base,
        type: "keyauth",
        keyauth: {
          keySpaceIds: values.keySpaceIds,
          ...(locations ? { locations } : {}),
          ...(values.permissionQuery ? { permissionQuery: values.permissionQuery } : {}),
        },
      };
    }
    case "ratelimit": {
      const key = (() => {
        switch (values.keySource) {
          case "remoteIp":
            return { remoteIp: {} };
          case "header":
            return { header: { name: values.keyValue } };
          case "authenticatedSubject":
            return { authenticatedSubject: {} };
          case "path":
            return { path: {} };
          case "principalClaim":
            return { principalClaim: { claimName: values.keyValue } };
        }
      })();

      return {
        ...base,
        type: "ratelimit",
        ratelimit: { limit: values.limit, windowMs: values.windowMs, key },
      };
    }
  }
}
