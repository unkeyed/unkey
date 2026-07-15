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
