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
 *   - transactions  → `event.request.url`, request headers (Referer),
 *                     transaction name, span descriptions, and span/trace
 *                     attributes such as `http.url`/`url.full`
 *   - spans         → standalone web-vital (INP) span envelopes, which bypass
 *                     `beforeSendTransaction` entirely
 *
 * This module redacts those values before the payload leaves the browser/server.
 * Error events run through `beforeSend`, transactions through
 * `beforeSendTransaction`, standalone spans through `beforeSendSpan`; the hooks
 * fire on disjoint payloads, so each family needs its own wiring in the Sentry
 * configs.
 */

import type { ErrorEvent, EventHint, NodeOptions } from "@sentry/nextjs";

/**
 * `@sentry/nextjs` does not re-export `TransactionEvent`, so derive it from
 * the `beforeSendTransaction` option this module's hook is wired into.
 */
export type TransactionEvent = Parameters<NonNullable<NodeOptions["beforeSendTransaction"]>>[0];

const REDACTED = "[REDACTED]";

/**
 * Names whose values are always secrets or PII, whether they appear as URL
 * query params or as tRPC input field names. Matched case-insensitively. Keep
 * this list aligned with the secrets we actually put into URLs across the
 * dashboard and auth flows.
 */
const SENSITIVE_NAME_KEYS = new Set(
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
 * Names treated as sensitive only when they appear as URL query params.
 * `input` carries tRPC GET batch payloads — JSON-encoded user data such as
 * emails — whose short values the token heuristic cannot catch, so it is
 * redacted wholesale in URLs. As a tRPC input *field* name it is benign, so
 * it stays out of the shared list above.
 */
const URL_ONLY_SENSITIVE_PARAM_KEYS = new Set(["input"]);

/**
 * Matches token-like substrings: 20+ chars of base64url/hex alphabet
 * containing at least one digit. This is a deliberately broad net to catch
 * opaque secrets (Unkey root keys, JWTs, session ids) that appear without a
 * recognizable parameter name — all of which mix digits into their alphabet.
 * The digit requirement spares long human-written identifiers like tRPC
 * procedure names (`getDeploymentRuntimeLogs`), which would otherwise collapse
 * to [REDACTED] and merge distinct routes in Sentry Performance. Only URL
 * scrubbing uses this net; free-standing values go through `redactTokenLike`.
 */
const TOKEN_LIKE = /(?=[A-Za-z0-9_-]*\d)[A-Za-z0-9_-]{20,}/g;

/**
 * Broad token-like matcher for free-standing values: the same alphabet as
 * `TOKEN_LIKE` but without the digit requirement. Field values carry no route
 * names, so there is no grouping to protect and this fails closed on pure-alpha
 * secrets (passphrases, digit-free ids) that the digit-required net spares.
 */
const BROAD_TOKEN_LIKE = /[A-Za-z0-9_-]{20,}/g;

/**
 * Whether a parameter/field name is known to carry secrets or PII. Matched
 * case-insensitively. Shared with the tRPC input scrubber in error-filter.ts so
 * both scrub surfaces treat the same names as sensitive.
 */
export function isSensitiveKey(name: string): boolean {
  return SENSITIVE_NAME_KEYS.has(name.toLowerCase());
}

/**
 * Redacts token-like substrings from a value regardless of its field name, as
 * a fail-closed fallback for opaque secrets under unrecognized names. Used by
 * the tRPC input scrubber in error-filter.ts.
 */
export function redactTokenLike(value: string): string {
  return value.replace(BROAD_TOKEN_LIKE, REDACTED);
}

/**
 * Whether a URL query parameter name is sensitive: the shared name list plus
 * the URL-only entries.
 */
function isSensitiveParamKey(name: string): boolean {
  const lower = name.toLowerCase();
  return SENSITIVE_NAME_KEYS.has(lower) || URL_ONLY_SENSITIVE_PARAM_KEYS.has(lower);
}

/**
 * Redacts the value of a single query parameter when its name is sensitive, and
 * otherwise redacts token-like values regardless of name. Returns the value to
 * store back into the query string.
 */
function scrubParamValue(name: string, value: string): string {
  if (isSensitiveParamKey(name)) {
    return REDACTED;
  }
  return value.replace(TOKEN_LIKE, REDACTED);
}

/**
 * Redacts every parameter value in place using `scrubParamValue`.
 */
function scrubSearchParams(params: URLSearchParams): void {
  for (const [name, value] of params.entries()) {
    params.set(name, scrubParamValue(name, value));
  }
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

    scrubSearchParams(parsed.searchParams);

    // Redact token-like segments embedded directly in the path. Next.js build
    // assets are exempt: their hashed chunk names look token-like but are
    // public, immutable files, and redacting them would make every resource
    // span indistinguishable in Sentry Performance.
    if (!parsed.pathname.startsWith("/_next/")) {
      parsed.pathname = parsed.pathname.replace(TOKEN_LIKE, REDACTED);
    }

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
 * Scrubs a `key=value&...` query string. A leading `?` is stripped by
 * `URLSearchParams`, matching how error-event query strings have always been
 * normalized; callers that need the `?` preserved re-attach it themselves.
 */
function scrubQueryParamsString(queryString: string): string {
  const params = new URLSearchParams(queryString);
  scrubSearchParams(params);
  return params.toString();
}

/**
 * Scrubs a raw query string (e.g. `event.request.query_string`), which Sentry
 * stores as either a `key=value&...` string or a record.
 */
function scrubQueryString(
  queryString: NonNullable<NonNullable<ErrorEvent["request"]>["query_string"]>,
): NonNullable<NonNullable<ErrorEvent["request"]>["query_string"]> {
  if (typeof queryString === "string") {
    return scrubQueryParamsString(queryString);
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
function scrubBreadcrumbs(event: ErrorEvent | TransactionEvent): void {
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
    scrubRequest(event.request);
    scrubBreadcrumbs(event);
  } catch {
    // Scrubbing must never prevent an error from being reported. If anything
    // unexpected happens, fall through and let Sentry send the event as-is.
  }
}

/**
 * Headers whose values are URLs and can carry secrets in their query strings,
 * e.g. the OAuth code in a `Referer` sent by a page loaded as
 * `/auth/callback?code=...`.
 */
const URL_HEADERS = new Set(["referer", "referrer", "location"]);

/**
 * Headers whose values are credentials and are redacted wholesale.
 * `sendDefaultPii: false` drops cookies but forwards other request headers
 * untouched on server/edge events.
 */
const SENSITIVE_HEADERS = new Set([
  "authorization",
  "proxy-authorization",
  "cookie",
  "set-cookie",
  "x-api-key",
]);

/**
 * Scrubs the URL, query string, and URL/credential-bearing headers of an
 * event's `request` payload in place.
 */
function scrubRequest(request: ErrorEvent["request"] | TransactionEvent["request"]): void {
  if (!request) {
    return;
  }
  if (typeof request.url === "string") {
    request.url = scrubUrl(request.url);
  }
  if (request.query_string != null) {
    request.query_string = scrubQueryString(request.query_string);
  }
  if (request.headers) {
    for (const [name, value] of Object.entries(request.headers)) {
      const lowerName = name.toLowerCase();
      if (SENSITIVE_HEADERS.has(lowerName)) {
        request.headers[name] = REDACTED;
      } else if (URL_HEADERS.has(lowerName) && typeof value === "string") {
        request.headers[name] = scrubUrl(value);
      }
    }
  }
}

/**
 * Span/trace attribute keys whose string values are URLs (absolute or
 * relative). Covers both the legacy `http.*` and current OpenTelemetry `url.*`
 * semantic conventions plus Next.js-specific attributes.
 */
const URL_ATTRIBUTE_KEYS = ["http.url", "url.full", "url", "http.target", "url.path", "next.url"];

/**
 * Span/trace attribute keys whose string values are raw query strings.
 */
const QUERY_ATTRIBUTE_KEYS = ["http.query", "url.query"];

/**
 * Span/trace attribute keys carrying URL fragments. Dropped entirely, matching
 * how `scrubUrl` discards fragments (e.g. the `/share#<id>` bearer id).
 */
const FRAGMENT_ATTRIBUTE_KEYS = ["http.fragment", "url.fragment"];

/**
 * Span/trace attribute keys carrying free-form text that may embed a URL,
 * e.g. the `transaction` route name that standalone web-vital (INP) spans
 * attach.
 */
const TEXT_ATTRIBUTE_KEYS = ["transaction"];

/**
 * Scrubs URL-carrying attributes on a span's `data` or a trace context's
 * `data` in place. Non-string values are left untouched.
 */
function scrubSpanAttributes(attributes: Record<string, unknown> | undefined): void {
  if (!attributes) {
    return;
  }
  for (const key of URL_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      attributes[key] = scrubUrl(value);
    }
  }
  for (const key of QUERY_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      // Browser tracing stores `http.query` with its leading `?`; keep the
      // attribute format stable.
      const scrubbed = scrubQueryParamsString(value);
      attributes[key] = value.startsWith("?") ? `?${scrubbed}` : scrubbed;
    }
  }
  for (const key of FRAGMENT_ATTRIBUTE_KEYS) {
    delete attributes[key];
  }
  for (const key of TEXT_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      attributes[key] = scrubUrlsInText(value);
    }
  }
}

/**
 * Scrubs URLs embedded in free-form text such as span descriptions
 * (`GET https://.../keys?token=x`, `GET /api/keys?key=x [200]`) and
 * transaction names (`/auth/callback`, `middleware GET /api/keys`). Works
 * token-by-token so text around the URL survives, and text without a
 * URL-shaped token (db/ui span descriptions, `trpc/key.create`-style names)
 * is returned unchanged.
 */
function scrubUrlsInText(text: string): string {
  return text
    .split(" ")
    .map((token) => {
      if (token.startsWith("/") || /^https?:\/\//i.test(token)) {
        return scrubUrl(token);
      }
      return token;
    })
    .join(" ");
}

/**
 * Scrubs PII/secrets from a transaction event in place and returns it, so it
 * can be wired directly as the `beforeSendTransaction` hook. Transactions and
 * their spans never pass through `beforeSend`, so without this hook the URLs
 * captured on traces (transaction `request.url`, `http.client` span
 * `http.url`/`url.full`, span descriptions) would leave the app unscrubbed —
 * the exact leak `scrubEventPii` closes for error events.
 *
 * Unlike `scrubEventPii`, this fails closed: if scrubbing throws partway
 * through, the transaction is dropped rather than sent half-scrubbed. Losing
 * a performance sample costs nothing; leaking a credential is unrecoverable.
 */
export function scrubTransactionPii(
  event: TransactionEvent,
  _hint?: EventHint,
): TransactionEvent | null {
  try {
    if (typeof event.transaction === "string") {
      event.transaction = scrubUrlsInText(event.transaction);
    }
    scrubRequest(event.request);
    scrubBreadcrumbs(event);

    // The root span's attributes live on the trace context, not in `spans`.
    scrubSpanAttributes(event.contexts?.trace?.data);

    for (const span of event.spans ?? []) {
      if (typeof span.description === "string") {
        span.description = scrubUrlsInText(span.description);
      }
      scrubSpanAttributes(span.data);
    }
    return event;
  } catch {
    return null;
  }
}

/**
 * `@sentry/nextjs` does not re-export `SpanJSON` either; derive it from the
 * `beforeSendSpan` option this module's span hook is wired into.
 */
export type SpanJson = Parameters<NonNullable<NodeOptions["beforeSendSpan"]>>[0];

/**
 * Scrubs PII/secrets from a single span, for the `beforeSendSpan` hook.
 * Standalone spans — the web-vital (INP) spans browser tracing emits by
 * default — are sent as their own envelopes and never pass through
 * `beforeSendTransaction`, so this hook is the only scrubbing they receive;
 * their page URL rides in the `transaction` attribute. The hook also re-runs
 * over each child span of a transaction, which is harmless because scrubbing
 * is idempotent. `beforeSendSpan` offers no drop path (the SDK requires the
 * span back), so on an unexpected error the span is returned as-is.
 */
export function scrubSpanPii(span: SpanJson): SpanJson {
  try {
    if (typeof span.description === "string") {
      span.description = scrubUrlsInText(span.description);
    }
    scrubSpanAttributes(span.data);
  } catch {
    // No drop path here — see above. Transaction-embedded spans still get a
    // second pass (and the fail-closed guard) in `scrubTransactionPii`.
  }
  return span;
}
