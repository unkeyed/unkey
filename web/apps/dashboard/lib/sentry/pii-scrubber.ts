/**
 * Sentry PII Scrubbing
 *
 * Centralized scrubbing of personally identifiable information and secrets from
 * Sentry payloads. `sendDefaultPii: false` stops Sentry from *adding* PII (IPs,
 * cookies, request bodies), but it does NOT scrub data that already lives inside
 * URLs we generate ourselves. Root keys, JWTs, OAuth codes and similar secrets
 * routinely appear in query strings and path segments, and those URLs surface in:
 *
 *   - error events  → `event.request.url`, `event.request.query_string`
 *   - breadcrumbs   → fetch/xhr `data.url`, navigation `data.from`/`data.to`
 *   - replay events → top-level `urls[]`
 *
 * This module redacts those values before the payload leaves the browser/server.
 */

import type { ErrorEvent, EventHint } from "@sentry/nextjs";

const REDACTED = "[REDACTED]";

/**
 * Query/path parameter names whose values are always secrets or PII. Matched
 * case-insensitively. Keep this list aligned with the secrets we actually put
 * into URLs across the dashboard and auth flows.
 */
const SENSITIVE_PARAM_KEYS = new Set(
  [
    "key",
    "apikey",
    "api_key",
    "rootkey",
    "root_key",
    "token",
    "access_token",
    "refresh_token",
    "id_token",
    "secret",
    "client_secret",
    "password",
    "pwd",
    "code",
    "state",
    "jwt",
    "authorization",
    "auth",
    "session",
    "email",
    "phone",
  ].map((k) => k.toLowerCase()),
);

/**
 * Matches token-like substrings: 20+ chars of base64url/hex alphabet. This is a
 * deliberately broad net to catch opaque secrets (Unkey root keys, JWTs,
 * session ids) that appear without a recognizable parameter name.
 */
const TOKEN_LIKE = /[A-Za-z0-9_-]{20,}/g;

/**
 * Redacts the value of a single query parameter when its name is sensitive, and
 * otherwise redacts token-like values regardless of name. Returns the value to
 * store back into the query string.
 */
function scrubParamValue(name: string, value: string): string {
  if (SENSITIVE_PARAM_KEYS.has(name.toLowerCase())) {
    return REDACTED;
  }
  return value.replace(TOKEN_LIKE, REDACTED);
}

/**
 * Scrubs secrets from a single URL (absolute or relative). Sensitive query
 * params are fully redacted, other params and the path have token-like segments
 * redacted. Returns the original string unchanged if it cannot be parsed so we
 * never throw inside a Sentry hook.
 */
export function scrubUrl(url: string): string {
  if (typeof url !== "string" || url.length === 0) {
    return url;
  }

  try {
    // Use a dummy base so relative URLs (the common case in breadcrumbs) parse.
    const base = "http://scrub.local";
    const parsed = new URL(url, base);

    for (const [name, value] of parsed.searchParams.entries()) {
      parsed.searchParams.set(name, scrubParamValue(name, value));
    }

    // Redact token-like segments embedded directly in the path.
    parsed.pathname = parsed.pathname.replace(TOKEN_LIKE, REDACTED);

    // Drop the fragment entirely. It is never useful for debugging and can carry
    // bearer credentials, e.g. the one-time share id in `/share#<id>` links.
    parsed.hash = "";

    const wasRelative = !/^[a-z][a-z0-9+.-]*:\/\//i.test(url);
    if (wasRelative) {
      // Reconstruct the relative form to avoid leaking the dummy origin.
      return `${parsed.pathname}${parsed.search}`;
    }
    return parsed.toString();
  } catch {
    // Fall back to a blanket token redaction if URL parsing fails.
    return url.replace(TOKEN_LIKE, REDACTED);
  }
}

/**
 * Scrubs a raw query string (e.g. `event.request.query_string`), which Sentry
 * stores as either a `key=value&...` string or a record.
 */
function scrubQueryString(
  queryString: NonNullable<NonNullable<ErrorEvent["request"]>["query_string"]>,
): NonNullable<NonNullable<ErrorEvent["request"]>["query_string"]> {
  if (typeof queryString === "string") {
    const params = new URLSearchParams(queryString);
    for (const [name, value] of params.entries()) {
      params.set(name, scrubParamValue(name, value));
    }
    return params.toString();
  }

  if (Array.isArray(queryString)) {
    return queryString.map(([name, value]) => [name, scrubParamValue(name, value)]) as Array<
      [string, string]
    >;
  }

  if (queryString && typeof queryString === "object") {
    const result: Record<string, string> = {};
    for (const [name, value] of Object.entries(queryString)) {
      result[name] = scrubParamValue(name, value);
    }
    return result;
  }

  return queryString;
}

/**
 * Scrubs URLs carried in breadcrumb data. Sentry's default fetch/xhr/navigation
 * breadcrumbs put URLs under `data.url`, `data.from`, and `data.to`.
 */
function scrubBreadcrumbs(event: ErrorEvent): void {
  if (!event.breadcrumbs) {
    return;
  }
  for (const breadcrumb of event.breadcrumbs) {
    const data = breadcrumb.data;
    if (!data) {
      continue;
    }
    for (const field of ["url", "from", "to"] as const) {
      const value = data[field];
      if (typeof value === "string") {
        data[field] = scrubUrl(value);
      }
    }
  }
}

/**
 * Scrubs PII/secrets from an error event in place. Safe to call on every event
 * regardless of classification. Never throws.
 *
 * @param event - The Sentry error event to scrub. Mutated in place because
 *   Sentry consumes the same object returned from `beforeSend`.
 */
export function scrubEventPii(event: ErrorEvent, _hint?: EventHint): void {
  try {
    if (event.request) {
      if (typeof event.request.url === "string") {
        event.request.url = scrubUrl(event.request.url);
      }
      if (event.request.query_string != null) {
        event.request.query_string = scrubQueryString(event.request.query_string);
      }
    }
    scrubBreadcrumbs(event);
  } catch {
    // Scrubbing must never prevent an error from being reported. If anything
    // unexpected happens, fall through and let Sentry send the event as-is.
  }
}
