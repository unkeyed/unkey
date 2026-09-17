import { logOperation } from "@/lib/logging";

type ManagedAuthEvent = "session_refresh" | "callback" | "widget_token" | "uncovered_route";

type ManagedAuthOutcome = "success" | "failure";

/**
 * Records a managed-auth outcome using fixed attributes.
 *
 * Provider errors, tokens, and user data must never be added here. `route` is
 * the only high-cardinality field: it belongs on the event so operators can see
 * which route produced the outcome, never in the message or a metric.
 */
export function logManagedAuthOutcome(
  event: ManagedAuthEvent,
  outcome: ManagedAuthOutcome,
  route?: string,
): void {
  logOperation(outcome === "failure" ? "warn" : "info", "Managed authentication outcome", {
    auth_event: event,
    auth_outcome: outcome,
    auth_route: route,
  });
}
