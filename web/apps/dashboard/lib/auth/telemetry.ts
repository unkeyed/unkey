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

/**
 * Records a request the middleware answered by redirecting to sign in because
 * AuthKit resolved no session. A caller that carried a session cookie lost a
 * session it previously had, which is the case worth surfacing; a caller
 * without one is an ordinary anonymous request.
 */
export function logUnauthenticatedRedirect(carriedSession: boolean): void {
  logOperation(carriedSession ? "warn" : "debug", "Unauthenticated request redirected to sign in", {
    auth_event: "unauthenticated_redirect",
    auth_outcome: "failure",
    auth_carried_session: carriedSession,
  });
}
