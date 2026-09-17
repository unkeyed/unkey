import { logOperation } from "@/lib/logging";

type ManagedAuthEvent = "session_refresh" | "callback" | "widget_token";

type ManagedAuthOutcome = "success" | "failure";

/**
 * Records a managed-auth outcome using fixed, low-cardinality attributes.
 *
 * Provider errors, URLs, tokens, and user data must never be added here.
 */
export function logManagedAuthOutcome(event: ManagedAuthEvent, outcome: ManagedAuthOutcome): void {
  logOperation(outcome === "failure" ? "warn" : "info", "Managed authentication outcome", {
    auth_event: event,
    auth_outcome: outcome,
  });
}
