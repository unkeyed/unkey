import { logOperation } from "@/lib/logging";

type ManagedAuthEvent = "session_refresh" | "callback" | "widget_token";

type ManagedAuthOutcome = "success" | "failure" | "slow";

/**
 * AuthKit validates the WorkOS session inline in the middleware and turns a
 * provider failure into an unauthenticated session without raising, so the only
 * app-side evidence of a provider stall is how long the call took. A normal
 * validation returns in a few hundred milliseconds; anything beyond this bound
 * is a stall worth a warn-level outcome.
 */
export const SLOW_SESSION_VALIDATION_MS = 5_000;

/**
 * Records a managed-auth outcome using fixed, low-cardinality attributes.
 *
 * Provider errors, URLs, tokens, and user data must never be added here.
 */
export function logManagedAuthOutcome(event: ManagedAuthEvent, outcome: ManagedAuthOutcome): void {
  logOperation(outcome === "success" ? "info" : "warn", "Managed authentication outcome", {
    auth_event: event,
    auth_outcome: outcome,
  });
}

export function logSessionValidationDuration(durationMs: number): void {
  if (durationMs >= SLOW_SESSION_VALIDATION_MS) {
    logManagedAuthOutcome("session_refresh", "slow");
  }
}
