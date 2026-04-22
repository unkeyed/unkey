import type { FormSelectOption } from "@unkey/ui";

// Single source of truth for grace-period choices. The server schema in
// `lib/trpc/routers/key/reroll.ts` validates against GRACE_PERIOD_VALUES_MS
// derived below, so adding or removing a row here updates both the form
// options and the API contract.
const GRACE_PERIODS = [
  { ms: 0, label: "Revoke immediately" },
  { ms: 60_000, label: "1 minute" },
  { ms: 900_000, label: "15 minutes" },
  { ms: 3_600_000, label: "1 hour" },
  { ms: 21_600_000, label: "6 hours" },
  { ms: 86_400_000, label: "24 hours" },
] as const;

export type GracePeriodMs = (typeof GRACE_PERIODS)[number]["ms"];

export const GRACE_PERIOD_VALUES_MS: readonly GracePeriodMs[] = GRACE_PERIODS.map((p) => p.ms);

export const GRACE_PERIOD_OPTIONS: FormSelectOption[] = GRACE_PERIODS.map((p) => ({
  value: String(p.ms),
  label: p.label,
}));

// Explicit literal so the default is independent of GRACE_PERIODS' order;
// the type annotation forces it to be one of the allowed values.
const DEFAULT_GRACE_PERIOD_MS: GracePeriodMs = 86_400_000;
export const DEFAULT_GRACE_PERIOD = String(DEFAULT_GRACE_PERIOD_MS);
