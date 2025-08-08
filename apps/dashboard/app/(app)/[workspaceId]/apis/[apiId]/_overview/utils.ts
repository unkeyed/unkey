import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import {
  type KeyVerificationOutcome,
  OUTCOME_BACKGROUND_COLORS,
  OUTCOME_BADGE_STYLES,
} from "./constants";

/**
 * Format an outcome string for display
 * (e.g., "RATE_LIMITED" -> "Rate Limited")
 */
export function formatOutcomeName(outcome: string): string {
  if (!outcome) {
    return "Unknown";
  }
  return outcome
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
}

/**
 * Get the background color class for an outcome
 */
export function getOutcomeColor(outcome: string): string {
  return (
    OUTCOME_BACKGROUND_COLORS[outcome as KeyVerificationOutcome] ||
    OUTCOME_BACKGROUND_COLORS.UNKNOWN
  );
}

/**
 * Get the badge style class for an outcome
 */
export function getOutcomeBadgeStyle(outcome: string): string {
  return OUTCOME_BADGE_STYLES[outcome as KeyVerificationOutcome] || OUTCOME_BADGE_STYLES.UNKNOWN;
}

/**
 * Get predefined outcome options for UI components
 */
export function getOutcomeOptions() {
  return KEY_VERIFICATION_OUTCOMES.filter((outcome) => outcome !== "") // Optionally exclude the empty case
    .map((outcome, index) => ({
      id: index + 1,
      outcome,
      display: formatOutcomeName(outcome),
      label: formatOutcomeName(outcome),
      color: getOutcomeColor(outcome),
      checked: false,
    }));
}
