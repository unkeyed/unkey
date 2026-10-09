import type { RateLimitIdentifier } from "@/lib/collections/deploy/policies.schema";
import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { plural } from "@/lib/fmt";
import { formatMs } from "@/lib/ms";
import { IconGaugeOutline18 } from "@unkey/icons";
import { P, match } from "@unkey/match";
import { z } from "zod";
import { sharedFormFields } from "../shared";
import type { PolicyKindModel } from "../types";
import { IDENTIFIER_SOURCES } from "./identifier-sources";

export const rateLimitIdentifierSourceSchema = z.enum([
  "remoteIp",
  "header",
  "authenticatedSubject",
  "path",
  "principalField",
]);
export type RateLimitIdentifierSource = z.infer<typeof rateLimitIdentifierSourceSchema>;

const ratelimitIdentifierRowSchema = z
  .object({
    source: rateLimitIdentifierSourceSchema,
    value: z.string(),
  })
  .superRefine((row, ctx) => {
    const { value } = IDENTIFIER_SOURCES[row.source];
    if (value && row.value.length === 0) {
      ctx.addIssue({ code: "custom", message: value.required, path: ["value"] });
    }
  });

export type RatelimitIdentifierRowValues = z.infer<typeof ratelimitIdentifierRowSchema>;

export const ratelimitFormSchema = z.object({
  ...sharedFormFields,
  type: z.literal("ratelimit"),
  limit: z.number().int().min(1, "Limit must be at least 1"),
  windowMs: z.number().int().min(1, "Window must be at least 1ms"),
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
    identifiers: [{ source: "remoteIp", value: "" }],
  }),
  toWire: (v) => ({
    type: "ratelimit",
    ratelimit: {
      limit: v.limit,
      windowMs: v.windowMs,
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
  summary: (v) =>
    `${plural(v.limit, "request")} per ${formatMs(v.windowMs)} · ${identifiersText(v.identifiers)}`,
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
  return match(key)
    .returnType<RatelimitIdentifierRowValues>()
    .with({ remoteIp: P._ }, () => ({ source: "remoteIp", value: "" }))
    .with({ header: P._ }, (k) => ({ source: "header", value: k.header.name }))
    .with({ authenticatedSubject: P._ }, () => ({ source: "authenticatedSubject", value: "" }))
    .with({ path: P._ }, () => ({ source: "path", value: "" }))
    .with({ principalField: P._ }, (k) => ({
      source: "principalField",
      value: k.principalField.path,
    }))
    .exhaustive();
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
