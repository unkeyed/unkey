import { OUTCOME_KINDS, type OutcomeCounts, type OutcomeKind } from "./schema/analytics.schema";

export const OUTCOME_LABELS: Record<OutcomeKind, string> = {
  rateLimited: "Rate limited",
  expired: "Expired",
  disabled: "Disabled",
  usageExceeded: "Usage exceeded",
  insufficientPermissions: "Insufficient permissions",
  forbidden: "Forbidden",
};

/** Filterable outcomes: VALID alongside the rejections, as on the dashboard. */
export const OUTCOME_FILTER_KINDS = ["valid", ...OUTCOME_KINDS] as const;
export type OutcomeFilterKind = (typeof OUTCOME_FILTER_KINDS)[number];

export const OUTCOME_FILTER_LABELS: Record<OutcomeFilterKind, string> = {
  valid: "Valid",
  ...OUTCOME_LABELS,
};

/**
 * The chart stacks one series per filterable outcome plus `other`, the
 * rejections the API counts in `total` without breaking them out.
 */
export const CHART_KINDS = [...OUTCOME_FILTER_KINDS, "other"] as const;
export type ChartKind = (typeof CHART_KINDS)[number];

export const CHART_LABELS: Record<ChartKind, string> = {
  ...OUTCOME_FILTER_LABELS,
  other: "Other",
};

/**
 * One colour per outcome, shared by the metrics strip and the filter rows. The
 * portal has no purple or blue token, so usage exceeded takes error-11.
 */
export const OUTCOME_COLORS: Record<ChartKind, string> = {
  valid: "hsl(var(--gray-8))",
  rateLimited: "hsl(var(--warning-9))",
  expired: "hsl(var(--orange-9))",
  disabled: "hsl(var(--gray-9))",
  usageExceeded: "hsl(var(--error-11))",
  insufficientPermissions: "hsl(var(--error-9))",
  forbidden: "hsl(var(--error-9))",
  other: "hsl(var(--gray-7))",
};

/**
 * Valid carries nearly all of the chart's area, so the bars fill in a far
 * lighter tint than the 8px swatch that labels them, which needs the contrast.
 */
export const CHART_FILLS: Record<ChartKind, string> = {
  ...OUTCOME_COLORS,
  valid: "hsl(var(--accent-4))",
};

export const VALID_COLOR = OUTCOME_COLORS.valid;

/** Every rejection together, as the metrics strip totals them. */
export const INVALID_COLOR = "hsl(var(--error-9))";

export function nonZeroOutcomes(
  outcomes: OutcomeCounts,
): Array<{ kind: OutcomeKind; count: number }> {
  return OUTCOME_KINDS.filter((kind) => outcomes[kind] > 0)
    .map((kind) => ({ kind, count: outcomes[kind] }))
    .sort((a, b) => b.count - a.count);
}
