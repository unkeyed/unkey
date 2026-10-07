import type { RateLimitIdentifier } from "@/lib/collections/deploy/policies.schema";
import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { formatMs } from "@/lib/ms";
import { IconGaugeOutline18 } from "@unkey/icons";
import { P, match } from "@unkey/match";
import { z } from "zod";
import { plural, sharedFormFields } from "../shared";
import type { KindFields, PolicyKindModel } from "../types";
import { IDENTIFIER_SOURCES } from "./identifier-sources";

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

export const ratelimitFormSchema = z.object({
  ...sharedFormFields,
  type: z.literal("ratelimit"),
  limit: z.number().int().min(1, "Limit must be at least 1"),
  windowMs: z.number().int().min(1, "Window must be at least 1ms"),
  // 2+ rows form a compound key where each unique combination of resolved
  // values gets its own counter.
  identifiers: z
    .array(ratelimitIdentifierRowSchema)
    .min(1, "Add at least one identifier")
    .max(POLICY_LIMITS.maxIdentifiersPerRatelimit),
});

export type RatelimitFormValues = z.infer<typeof ratelimitFormSchema>;

export const ratelimit: PolicyKindModel<"ratelimit", RatelimitFormValues> = {
  label: "Rate Limit",
  Icon: IconGaugeOutline18,
  does: "Counts requests by client IP, header, path or identity. To count by identity, put a Key Auth above it.",
  rejects: [{ status: 429, reason: "Over the limit, or the value to count by is missing" }],
  defaults: () => ({
    type: "ratelimit",
    limit: 100,
    windowMs: 60000,
    identifiers: [{ id: crypto.randomUUID(), source: "remoteIp", value: "" }],
  }),
  toWire: (v) => ({
    type: "ratelimit",
    ratelimit: {
      limit: v.limit,
      windowMs: v.windowMs,
      // Always serialize the identifiers array, even for one row, so all
      // writes converge on the target shape. Deserialization still reads
      // the deprecated single identifier from old stored policies.
      identifiers: v.identifiers.map((row) => toRateLimitIdentifier(row.source, row.value)),
    },
  }),
  fromWire: (p) => ({
    type: "ratelimit",
    limit: p.ratelimit.limit,
    windowMs: p.ratelimit.windowMs,
    identifiers: (
      p.ratelimit.identifiers ?? (p.ratelimit.identifier ? [p.ratelimit.identifier] : [])
    ).map(fromRateLimitIdentifier),
  }),
};

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

function fromRateLimitIdentifier(key: RateLimitIdentifier): RatelimitIdentifierRowValues {
  const id = crypto.randomUUID();
  return match(key)
    .returnType<RatelimitIdentifierRowValues>()
    .with({ remoteIp: P._ }, () => ({ id, source: "remoteIp", value: "" }))
    .with({ header: P._ }, (k) => ({ id, source: "header", value: k.header.name }))
    .with({ authenticatedSubject: P._ }, () => ({ id, source: "authenticatedSubject", value: "" }))
    .with({ path: P._ }, () => ({ id, source: "path", value: "" }))
    .with({ principalField: P._ }, (k) => ({
      id,
      source: "principalField",
      value: k.principalField.path,
    }))
    .exhaustive();
}

export function ratelimitSummary(
  v: Pick<KindFields<RatelimitFormValues>, "limit" | "windowMs" | "identifiers">,
): string {
  return `${plural(v.limit, "request")} per ${formatMs(v.windowMs)} · ${identifiersText(v.identifiers)}`;
}

function identifiersText(rows: RatelimitIdentifierRowValues[]): string {
  if (rows.length === 0) {
    return "no identifier";
  }
  return `per ${rows
    .map((row) => {
      const { short } = IDENTIFIER_SOURCES[row.source];
      return row.value ? `${short} ${row.value}` : short;
    })
    .join(" + ")}`;
}
