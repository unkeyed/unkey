import type { FormSelectOption } from "@unkey/ui";

// Single source of truth for grace-period choices. The server schema in
// `lib/trpc/routers/key/reroll/index.ts` validates against
// GRACE_PERIOD_VALUES_MS derived below, so adding or removing a row here
// updates both the form options and the API contract.
const GRACE_PERIODS = [
  { ms: 0, label: "Revoke immediately" },
  { ms: 60_000, label: "1 minute" },
  { ms: 900_000, label: "15 minutes" },
  { ms: 3_600_000, label: "1 hour" },
  { ms: 21_600_000, label: "6 hours" },
  { ms: 86_400_000, label: "24 hours" },
] as const;

export type GracePeriodMs = (typeof GRACE_PERIODS)[number]["ms"];

// Stringified form of GracePeriodMs. The FormSelect deals in strings, so
// the form schema narrows to this union and converts back via
// `gracePeriodMsFromValue` at the API boundary.
export type GracePeriodValue = `${GracePeriodMs}`;

export const GRACE_PERIOD_VALUES_MS: readonly GracePeriodMs[] = GRACE_PERIODS.map((p) => p.ms);

export const GRACE_PERIOD_OPTIONS: FormSelectOption[] = GRACE_PERIODS.map((p) => ({
  value: String(p.ms),
  label: p.label,
}));

// Lookup from form-select string to typed numeric ms. Built from the same
// table so the conversion needs no cast at the call site — the mapped type
// proves the result is a GracePeriodMs that exactly matches the key.
const MS_BY_VALUE = Object.fromEntries(GRACE_PERIODS.map((p) => [String(p.ms), p.ms])) as {
  readonly [K in GracePeriodMs as `${K}`]: K;
};

export function isGracePeriodValue(value: string): value is GracePeriodValue {
  return value in MS_BY_VALUE;
}

export function gracePeriodMsFromValue<V extends GracePeriodValue>(
  value: V,
): (typeof MS_BY_VALUE)[V] {
  return MS_BY_VALUE[value];
}

// Explicit literal so the default is independent of GRACE_PERIODS' order;
// the type annotation forces it to be one of the allowed values.
const DEFAULT_GRACE_PERIOD_MS: GracePeriodMs = 86_400_000;
export const DEFAULT_GRACE_PERIOD: GracePeriodValue = `${DEFAULT_GRACE_PERIOD_MS}`;
