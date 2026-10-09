import {
  type MatchExpr,
  POLICY_LIMITS,
  type StringMatch,
  type StringMatchMode,
  httpMethodSchema,
  matchExprSchema,
  stringMatchModeSchema,
} from "@/lib/collections/deploy/policies.schema";
import { P, match } from "@unkey/match";
import { z } from "zod";

const namedOperatorSchema = z.enum([...stringMatchModeSchema.options, "present"]);

const pathConditionSchema = z.object({
  type: z.literal("path"),
  operator: stringMatchModeSchema,
  // Canonical stringMatchValue is min(1) — enforce here so users see a
  // field-level error instead of a generic 500 from savePolicies.
  value: z.string().min(1, "Value is required"),
});

const methodConditionSchema = z.object({
  type: z.literal("method"),
  operator: z.literal("anyOf"),
  methods: z.array(httpMethodSchema).min(1, "Select at least one method"),
});

const headerConditionSchema = z.object({
  type: z.literal("header"),
  name: z.string().min(1, "Header name is required"),
  operator: namedOperatorSchema,
  value: z.string(),
});

const queryParamConditionSchema = z.object({
  type: z.literal("queryParam"),
  name: z.string().min(1, "Param name is required"),
  operator: namedOperatorSchema,
  value: z.string(),
});

const remoteIpConditionSchema = z.object({
  type: z.literal("remoteIp"),
  operator: z.enum(["in", "notIn"]),
  value: z.string(),
});

const ipOrCidrSchema = z.union([z.ipv4(), z.ipv6(), z.cidrv4(), z.cidrv6()]);

function splitRanges(ranges: string): string[] {
  return ranges.split(/[\s,]+/).filter((r) => r.length > 0);
}

export const matchConditionSchema = z
  .discriminatedUnion("type", [
    pathConditionSchema,
    methodConditionSchema,
    headerConditionSchema,
    queryParamConditionSchema,
    remoteIpConditionSchema,
  ])
  .superRefine((c, ctx) => {
    if (c.type === "remoteIp") {
      const entries = splitRanges(c.value);
      const invalid = entries.find((entry) => !ipOrCidrSchema.safeParse(entry).success);
      const message = match({ count: entries.length, invalid })
        .with({ count: 0 }, () => "Enter at least one IP address or CIDR")
        .when(
          ({ count }) => count > POLICY_LIMITS.maxRemoteIpEntries,
          () => `At most ${POLICY_LIMITS.maxRemoteIpEntries} ranges`,
        )
        .with({ invalid: P.string }, (v) => `${v.invalid} is not a valid IP address or CIDR`)
        .otherwise(() => null);
      if (message !== null) {
        ctx.addIssue({ code: "custom", message, path: ["value"] });
      }
    }
    // A presence check sends no value; every other named match sends a
    // stringMatch, which the API rejects when empty.
    if ("name" in c && c.operator !== "present" && !c.value) {
      ctx.addIssue({ code: "custom", message: "Value is required", path: ["value"] });
    }
  });

export type MatchConditionFormValues = z.infer<typeof matchConditionSchema>;
export type ConditionType = MatchConditionFormValues["type"];
export type ConditionOf<T extends ConditionType> = Extract<MatchConditionFormValues, { type: T }>;
export type ConditionOperator = MatchConditionFormValues["operator"];

function toStringMatch(mode: StringMatchMode, value: string): StringMatch {
  return match(mode)
    .returnType<StringMatch>()
    .with("exact", () => ({ exact: value }))
    .with("prefix", () => ({ prefix: value }))
    .with("regex", () => ({ regex: value }))
    .exhaustive();
}

export function toMatchExpr(condition: MatchConditionFormValues): MatchExpr {
  return match(condition)
    .returnType<MatchExpr>()
    .with({ type: "path" }, (c) => ({ path: { path: toStringMatch(c.operator, c.value) } }))
    .with({ type: "method" }, (c) => ({ method: { methods: c.methods } }))
    .with({ type: "header" }, (c) =>
      c.operator === "present"
        ? { header: { name: c.name, present: true } }
        : { header: { name: c.name, value: toStringMatch(c.operator, c.value) } },
    )
    .with({ type: "queryParam" }, (c) =>
      c.operator === "present"
        ? { queryParam: { name: c.name, present: true } }
        : { queryParam: { name: c.name, value: toStringMatch(c.operator, c.value) } },
    )
    .with({ type: "remoteIp" }, (c) =>
      c.operator === "in"
        ? { remoteIp: { in: splitRanges(c.value) } }
        : { remoteIp: { notIn: splitRanges(c.value) } },
    )
    .exhaustive();
}

function fromStringMatch(sm: StringMatch): { operator: StringMatchMode; value: string } {
  return match(sm)
    .returnType<{ operator: StringMatchMode; value: string }>()
    .with({ exact: P.string }, (s) => ({ operator: "exact", value: s.exact }))
    .with({ prefix: P.string }, (s) => ({ operator: "prefix", value: s.prefix }))
    .with({ regex: P.string }, (s) => ({ operator: "regex", value: s.regex }))
    .exhaustive();
}

const PRESENT = { operator: "present", value: "" } as const;

export function fromMatchExpr(raw: unknown): MatchConditionFormValues | null {
  const parsed = matchExprSchema.safeParse(raw);
  if (!parsed.success) {
    return null;
  }
  return match(parsed.data)
    .returnType<MatchConditionFormValues>()
    .with({ path: P._ }, (e) => ({ type: "path", ...fromStringMatch(e.path.path) }))
    .with({ method: P._ }, (e) => ({
      type: "method",
      operator: "anyOf",
      methods: e.method.methods,
    }))
    .with({ header: P._ }, (e) => ({
      type: "header",
      name: e.header.name,
      ...("present" in e.header ? PRESENT : fromStringMatch(e.header.value)),
    }))
    .with({ queryParam: P._ }, (e) => ({
      type: "queryParam",
      name: e.queryParam.name,
      ...("present" in e.queryParam ? PRESENT : fromStringMatch(e.queryParam.value)),
    }))
    .with({ remoteIp: P._ }, (e) =>
      "in" in e.remoteIp
        ? { type: "remoteIp", operator: "in", value: e.remoteIp.in.join(", ") }
        : { type: "remoteIp", operator: "notIn", value: e.remoteIp.notIn.join(", ") },
    )
    .exhaustive();
}
