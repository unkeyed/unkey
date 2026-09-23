import type { KeyVerificationOutcome } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/constants";
import { formatOutcomeName } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/utils";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";

export function rampColorVar(rampStep: string): string {
  return `var(--color-${rampStep})`;
}

// VALID is deliberately the neutral baseline the stacked bar rests on, not the
// success-9 that OUTCOME_BACKGROUND_COLORS uses for filter chips.
const OUTCOME_RAMP_STEP: Record<KeyVerificationOutcome, string> = {
  VALID: "gray-4",
  RATE_LIMITED: "warning-9",
  INSUFFICIENT_PERMISSIONS: "error-9",
  FORBIDDEN: "error-9",
  DISABLED: "gray-9",
  EXPIRED: "orange-9",
  USAGE_EXCEEDED: "feature-9",
  "": "gray-9",
};

export function outcomeChartColor(outcome: KeyVerificationOutcome): string {
  return rampColorVar(OUTCOME_RAMP_STEP[outcome]);
}

export function createOutcomeChartConfig(
  includedOutcomes?: KeyVerificationOutcome[],
): Record<string, { label: string; color: string }> {
  const config: Record<string, { label: string; color: string }> = {
    success: {
      label: formatOutcomeName("VALID"),
      color: outcomeChartColor("VALID"),
    },
  };

  const outcomesToInclude =
    includedOutcomes ??
    KEY_VERIFICATION_OUTCOMES.filter((outcome) => outcome !== "VALID" && outcome !== "");

  for (const outcome of outcomesToInclude) {
    if (outcome === "VALID" || outcome === "") {
      continue;
    }
    config[outcome.toLowerCase()] = {
      label: formatOutcomeName(outcome),
      color: outcomeChartColor(outcome),
    };
  }

  return config;
}
