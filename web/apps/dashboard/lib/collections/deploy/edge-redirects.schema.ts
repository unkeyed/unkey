/**
 * Edge-redirect schema — single source of truth for the dashboard.
 *
 * Wire shape vs. Go service:
 *   The Go frontline service (svc/frontline/services/edgeredirect) parses these
 *   blobs as protojson with `DiscardUnknown: true`. Each Rule has a oneof on
 *   `requireHttps | stripWww | addWww | hostRewrite` (camelCase from the
 *   underlying snake_case proto fields). We persist the protojson directly so
 *   no transformation is needed on the Go side.
 *
 *   The dashboard surfaces a single www-handling toggle that operates on the
 *   apex/www domain *pair*, not a single row: picking "redirect www → apex"
 *   writes a StripWWW rule on the www row's config and clears the apex row's
 *   config. Picking "redirect apex → www" does the inverse. See deriveWwwMode
 *   for the read direction. HostRewrite is supported by the engine but has
 *   no UI yet.
 */
import { z } from "zod";

// ── Wire shape (matches frontline.edgeredirect.v1.Config protojson) ─────

const ruleBase = {
  id: z.string().min(1),
  enabled: z.boolean(),
  status: z.union([z.literal(301), z.literal(308)]).optional(),
} as const;

const hostnameSchema = z
  .string()
  .min(1)
  .max(253)
  .regex(/^[a-zA-Z0-9.-]+$/, "Hostname must contain only letters, digits, dots, and hyphens");

export const wireRuleSchema = z.union([
  z.object({ ...ruleBase, requireHttps: z.object({}).strict() }).strict(),
  z.object({ ...ruleBase, stripWww: z.object({}).strict() }).strict(),
  z.object({ ...ruleBase, addWww: z.object({}).strict() }).strict(),
  z
    .object({
      ...ruleBase,
      hostRewrite: z.object({ from: hostnameSchema, to: hostnameSchema }).strict(),
    })
    .strict(),
]);
export type WireRule = z.infer<typeof wireRuleSchema>;

export const wireConfigSchema = z.object({
  rules: z.array(wireRuleSchema),
});
export type WireConfig = z.infer<typeof wireConfigSchema>;

// ── Form shape (what the UI binds to) ────────────────────────────────────

export const wwwModeSchema = z.enum(["none", "stripWww", "addWww"]);
export type WwwMode = z.infer<typeof wwwModeSchema>;

// ── Pair-aware helpers ───────────────────────────────────────────────────

/**
 * Splits a domain into its apex/www pair. Idempotent: passing either form
 * yields the same pair. The www variant is always returned even for
 * single-label inputs (e.g. "localhost") so callers can attempt lookup;
 * pair-mode toggling on those is not meaningful and the UI hides it.
 */
export function domainPair(domain: string): { apex: string; www: string } {
  const trimmed = domain.trim().toLowerCase();
  if (trimmed.startsWith("www.")) {
    return { apex: trimmed.slice(4), www: trimmed };
  }
  return { apex: trimmed, www: `www.${trimmed}` };
}

/**
 * Reads the joint www-handling mode from the two rows' parsed configs.
 * Either side may be null when that row does not exist (custom_domain
 * registered but no frontline_route yet). The first matching enabled rule
 * wins on each side; mixed / unexpected combinations resolve to "none".
 */
export function deriveWwwMode(
  apexConfig: WireConfig | null,
  wwwConfig: WireConfig | null,
): WwwMode {
  const wwwHasStrip = wwwConfig?.rules.some((r) => r.enabled && "stripWww" in r);
  if (wwwHasStrip) {
    return "stripWww";
  }
  const apexHasAdd = apexConfig?.rules.some((r) => r.enabled && "addWww" in r);
  if (apexHasAdd) {
    return "addWww";
  }
  return "none";
}

/**
 * Returns the wire config to write to each row of the pair given a target
 * mode. Always writes both sides — even the "no rule" side — so toggling
 * cleanly reverts whichever direction was previously configured.
 */
export function pairConfigForMode(mode: WwwMode): {
  apex: WireConfig;
  www: WireConfig;
} {
  switch (mode) {
    case "stripWww":
      return {
        apex: { rules: [] },
        www: { rules: [{ id: "strip-www", enabled: true, stripWww: {} }] },
      };
    case "addWww":
      return {
        apex: { rules: [{ id: "add-www", enabled: true, addWww: {} }] },
        www: { rules: [] },
      };
    case "none":
      return { apex: { rules: [] }, www: { rules: [] } };
  }
}
