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
 * Records a request that reached a page route without the AuthKit middleware
 * headers. The concrete path is not available, because the middleware is what
 * stamps it on the request; matched route, host, and client identify it instead,
 * and the accept header separates a document request from a subresource probe.
 */
export function logAuthkitMiddlewareBypass(requestHeaders: Headers): void {
  logOperation("warn", "Request bypassed the AuthKit middleware", {
    auth_event: "middleware_bypass",
    next_route: requestHeaders.get("x-matched-path") ?? undefined,
    request_host: requestHeaders.get("host") ?? undefined,
    request_user_agent: requestHeaders.get("user-agent") ?? undefined,
    request_accept: requestHeaders.get("accept") ?? undefined,
  });
}
