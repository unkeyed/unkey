import type { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";

/**
 * Type for key verification outcomes
 */
export type KeyVerificationOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];

/**
 * Background color classes for each outcome
 */
export const OUTCOME_BACKGROUND_COLORS: Record<string, string> = {
  VALID: "bg-success-9",
  RATE_LIMITED: "bg-warning-9",
  INSUFFICIENT_PERMISSIONS: "bg-error-9",
  FORBIDDEN: "bg-error-9",
  DISABLED: "bg-gray-9",
  EXPIRED: "bg-orange-9",
  USAGE_EXCEEDED: "bg-feature-9",
  "": "bg-accent-9",
};

/**
 * Badge style classes for each outcome
 */
export const OUTCOME_BADGE_STYLES: Record<string, string> = {
  VALID: "bg-gray-4 text-accent-11 hover:bg-gray-5 group-hover:text-accent-12",
  RATE_LIMITED: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
  INSUFFICIENT_PERMISSIONS: "bg-error-4 text-error-11 group-hover:bg-error-5",
  FORBIDDEN: "bg-error-4 text-error-11 group-hover:bg-error-5",
  DISABLED: "bg-gray-4 text-gray-11 group-hover:bg-gray-5",
  EXPIRED: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
  USAGE_EXCEEDED: "bg-feature-4 text-feature-11 group-hover:bg-feature-5",
  "": "bg-gray-4 text-accent-11 hover:bg-gray-5 group-hover:text-accent-12", // Empty string case
};
