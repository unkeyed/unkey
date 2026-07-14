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
 * Beyond URLs, error and transaction events carry the raw tRPC procedure input
 * under `contexts.trpc.input` (`attachRpcInput: true`), scrubbed by
 * `scrubTrpcInput`, and server db spans record interpolated SQL statements
 * under `db.query.text`, masked by `maskSqlLiterals`.
 *
 * This module redacts those values before the payload leaves the browser/server.
 * Error events run through `beforeSend`, transactions through
 * `beforeSendTransaction`, standalone spans through `beforeSendSpan`; the hooks
 * fire on disjoint payloads, so each family needs its own wiring in the Sentry
 * configs.
 */

import type { ErrorEvent, EventHint, NodeOptions } from "@sentry/nextjs";
import type { Router } from "../trpc/routers";

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
 * Matches token-like runs: 20+ chars of base64url alphabet. `redactTokenLike`
 * redacts every match; `redactTokenLikeInPath` additionally requires a digit
 * before redacting.
 */
const TOKEN_LIKE = /[A-Za-z0-9_-]{20,}/g;

/**
 * Whether a parameter/field name is known to carry secrets or PII. Matched
 * case-insensitively. Shared by the URL param scrubber and the tRPC input
 * scrubber so both surfaces treat the same names as sensitive.
 */
function isSensitiveKey(name: string): boolean {
  return SENSITIVE_NAME_KEYS.has(name.toLowerCase());
}

/**
 * Redacts token-like substrings from a value regardless of its field name, as
 * a fail-closed fallback for opaque secrets under unrecognized names. Applied
 * everywhere route grouping is not a concern — query param values, tRPC input
 * field values, unparseable URLs — so digit-free secrets (passphrases,
 * letters-only ids) fail closed.
 */
function redactTokenLike(value: string): string {
  return value.replace(TOKEN_LIKE, REDACTED);
}

/**
 * Redacts token-like path segments only when the run contains a digit. Opaque
 * secrets (Unkey root keys, JWTs, session ids) mix digits into their alphabet;
 * the digit requirement spares long human-written route identifiers like tRPC
 * procedure names (`getDeploymentRuntimeLogs`), which would otherwise collapse
 * to [REDACTED] and merge distinct routes in Sentry Performance. Only URL
 * paths get this treatment — route grouping lives in paths — accepting a
 * known residual miss: a digit-free 20+ char secret embedded directly in a
 * path segment. The digit check runs per matched run rather than as a regex
 * lookahead; a lookahead re-scans from every position, turning long
 * digit-free paths into an ~O(n²) stall on the browser main thread.
 */
function redactTokenLikeInPath(path: string): string {
  return path.replace(TOKEN_LIKE, (run) => (/\d/.test(run) ? REDACTED : run));
}

/**
 * Whether a URL query parameter name is sensitive: the shared name list plus
 * the URL-only entries.
 */
function isSensitiveParamKey(name: string): boolean {
  const lower = name.toLowerCase();
  // ClickHouse sends query bindings as `param_<name>` search params; bindings
  // are data values by definition (external ids are often emails), never
  // route info, so redact them regardless of the bound name.
  if (lower.startsWith("param_")) {
    return true;
  }
  return isSensitiveKey(name) || URL_ONLY_SENSITIVE_PARAM_KEYS.has(lower);
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
  return redactTokenLike(value);
}

/**
 * Redacts every parameter value in place using `scrubParamValue`. Works on a
 * snapshot of the entries: calling `set()` while iterating collapses repeated
 * keys (`?status=a&status=b` would silently lose values), so entries are
 * removed and re-appended scrubbed, preserving duplicates and order.
 * Repeated deletes of the same name are no-ops.
 */
function scrubSearchParams(params: URLSearchParams): void {
  const entries = [...params.entries()];
  for (const [name] of entries) {
    params.delete(name);
  }
  for (const [name, value] of entries) {
    params.append(name, scrubParamValue(name, value));
  }
}

/**
 * Scrubs secrets from a single URL (absolute or relative). Sensitive query
 * params are fully redacted, other param values have broad token-like runs
 * redacted, path segments have digit-bearing token-like runs redacted,
 * basic-auth userinfo and fragments are dropped, and non-http(s) opaque-path
 * schemes (mailto:, tel:, blob:) are redacted wholesale down to the scheme.
 * Never throws inside a Sentry hook: if the URL cannot be parsed, the raw
 * string gets a blanket broad-net redaction instead.
 */
export function scrubUrl(url: string): string {
  if (typeof url !== "string" || url.length === 0) {
    return url;
  }

  // Opaque-path schemes (mailto:, tel:, blob:, data:) never reach the
  // path/query machinery — the WHATWG pathname setter is a no-op on opaque
  // paths — and their payload is inherently identifying (an email, a phone
  // number, a blob id). Redact the payload wholesale, keeping the scheme.
  const opaqueScheme = url.match(/^(?!https?:)([a-z][a-z0-9+.-]*):(?!\/\/)/i)?.[1];
  if (opaqueScheme) {
    return `${opaqueScheme}:${REDACTED}`;
  }

  try {
    // Use a dummy base so relative URLs (the common case in breadcrumbs) parse.
    const base = "http://scrub.local";
    const parsed = new URL(url, base);

    // Basic-auth userinfo is always a credential; drop it outright.
    parsed.username = "";
    parsed.password = "";

    scrubSearchParams(parsed.searchParams);

    // Redact token-like segments embedded directly in the path. Next.js static
    // build assets are exempt: their hashed chunk names look token-like but
    // are public, immutable files, and redacting them would make every
    // resource span indistinguishable in Sentry Performance. Only
    // `/_next/static/` is exempt — `/_next/data/` payload URLs embed dynamic
    // route params, which can be token-like ids.
    if (!parsed.pathname.startsWith("/_next/static/")) {
      parsed.pathname = redactTokenLikeInPath(parsed.pathname);
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
    // Fall back to a blanket broad-net redaction if URL parsing fails; with
    // no parsed structure there is no route grouping to protect.
    return redactTokenLike(url);
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
    return queryString.map(([name, value]): [string, string] => [
      name,
      scrubParamValue(name, value),
    ]);
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
 * Scrubs PII/secrets from an error event in place: request URLs and headers,
 * breadcrumbs, and the attached tRPC input. Safe to call on every event
 * regardless of classification. Never throws.
 *
 * @param event - The Sentry error event to scrub. Mutated in place because
 *   Sentry consumes the same object returned from `beforeSend`.
 */
export function scrubEventPii(event: ErrorEvent, _hint?: EventHint): void {
  try {
    // The credential-bearing tRPC input is scrubbed FIRST (and is internally
    // fail-closed): if URL scrubbing below throws, the outer catch sends the
    // event as-is, and by then the input must already be redacted.
    scrubTrpcInput(event);
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
 * Input field names that may carry plaintext secrets and must never be sent
 * to Sentry, beyond the shared sensitive-name list (which already covers
 * `secret`, `code`, `token`, `password`, etc.). The tRPC Sentry middleware
 * attaches the raw procedure input to `event.contexts.trpc.input` (via
 * `attachRpcInput: true`), which would otherwise exfiltrate env-var secret
 * values (`value`, `variables[].value`, `items[].value`).
 * `sendDefaultPii: false` does not scrub custom contexts, so we redact these
 * explicitly. Matched case-insensitively.
 */
const SENSITIVE_INPUT_KEYS = new Set(["value"]);

/**
 * Whether a tRPC input field name must be redacted: the shared sensitive-name
 * list plus input-specific names.
 */
function isSensitiveInputKey(key: string): boolean {
  return SENSITIVE_INPUT_KEYS.has(key.toLowerCase()) || isSensitiveKey(key);
}

/**
 * Dotted paths of the `share` router's procedures, derived from the app router
 * type (type-only import, erased at runtime). Typing the set below against
 * this union makes a procedure rename a compile error instead of a silently
 * disabled redaction.
 */
type ShareProcedurePath = `share.${Extract<keyof Router["share"]["_def"]["record"], string>}`;

/**
 * Procedures whose input is itself a bearer credential, so key-based scrubbing
 * is not enough and the entire input must be redacted. `share.reveal` takes
 * `{ id }` where the id is the one-time share credential: an unexpected error
 * (e.g. a vault outage, which rolls back the transaction and leaves the row
 * un-consumed) would otherwise store a still-valid credential in Sentry.
 */
const CREDENTIAL_INPUT_PROCEDURES: ReadonlySet<string> = new Set<ShareProcedurePath>([
  "share.reveal",
]);

/**
 * Recursively replaces the values of sensitive keys with a redaction marker.
 * Walks nested objects and arrays so secrets nested under `variables`/`items`
 * are also scrubbed, and redacts token-like substrings from the remaining
 * string values as a fail-closed fallback for secrets under field names the
 * key list doesn't anticipate. Returns a new structure and never mutates the
 * input.
 */
function redactSensitiveValues(value: unknown): unknown {
  if (typeof value === "string") {
    return redactTokenLike(value);
  }

  if (Array.isArray(value)) {
    return value.map(redactSensitiveValues);
  }

  if (value && typeof value === "object") {
    const result: Record<string, unknown> = {};
    for (const [key, nested] of Object.entries(value as Record<string, unknown>)) {
      result[key] = isSensitiveInputKey(key) ? REDACTED : redactSensitiveValues(nested);
    }
    return result;
  }

  return value;
}

/**
 * Scrubs plaintext secrets from the tRPC input attached by the Sentry tRPC
 * middleware. The middleware writes to the scope, and scope contexts merge
 * into transaction events just as they do into error events, so this runs in
 * both `scrubEventPii` and `scrubTransactionPii` — tRPC-named transactions
 * sample at 100%, so an unscrubbed transaction would store e.g.
 * `share.reveal`'s one-time bearer id on every successful reveal. Fails
 * closed for this credential-bearing surface: if scrubbing throws, the input
 * is dropped wholesale rather than forwarded raw. Mutates the event in place
 * because Sentry consumes the same object returned from its hooks.
 */
function scrubTrpcInput(event: ErrorEvent | TransactionEvent): void {
  const trpcContext = event.contexts?.trpc;
  if (!trpcContext || !("input" in trpcContext)) {
    return;
  }

  try {
    const procedurePath = trpcContext.procedure_path;
    if (typeof procedurePath === "string" && CREDENTIAL_INPUT_PROCEDURES.has(procedurePath)) {
      trpcContext.input = REDACTED;
      return;
    }

    trpcContext.input = redactSensitiveValues(trpcContext.input);
  } catch {
    trpcContext.input = REDACTED;
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
 * Span attribute keys carrying the executed SQL statement. The mysql2
 * OpenTelemetry auto-instrumentation records statements with bound values
 * interpolated as literals (its `maskStatement` option defaults to false), so
 * WHERE/INSERT literals — share ids, emails, key ids — would otherwise reach
 * Sentry on every db span. `db.query.text` is the current semantic
 * convention, `db.statement` the legacy one.
 */
const SQL_ATTRIBUTE_KEYS = ["db.query.text", "db.statement"];

/**
 * Every attribute key the span scrubber touches, in one list. The fail-closed
 * catch in `scrubSpanPii` deletes exactly these keys, so a new key group only
 * needs to be registered here once, next to its `scrubSpanAttributes`
 * handling — the scrub list and the fallback deletion list cannot drift.
 */
const SCRUBBED_ATTRIBUTE_KEYS = [
  ...URL_ATTRIBUTE_KEYS,
  ...QUERY_ATTRIBUTE_KEYS,
  ...FRAGMENT_ATTRIBUTE_KEYS,
  ...TEXT_ATTRIBUTE_KEYS,
  ...SQL_ATTRIBUTE_KEYS,
];

/**
 * Masks literal values in a SQL statement while keeping its shape: quoted
 * strings and standalone numeric literals become `?`, matching what the
 * instrumentation's `maskStatement` option would produce, so keywords,
 * identifiers, and query grouping survive. The trailing SQLCommenter block is
 * preserved wholesale — its quoted tag values carry no user data and Query
 * Insights attribution depends on them.
 *
 * The trailer is peeled off with a linear scan — an end-anchored regex with a
 * greedy prefix backtracks quadratically on statements full of comment
 * openers — and bound to the LAST `/*`, so an opener inside a user-supplied
 * value cannot fake a comment and smuggle the surrounding literals through
 * unmasked; string literals are then masked first over the body.
 */
function maskSqlLiterals(sql: string): string {
  let body = sql;
  let trailer = "";
  if (sql.trimEnd().endsWith("*/")) {
    const openIndex = sql.lastIndexOf("/*");
    if (openIndex >= 0) {
      body = sql.slice(0, openIndex);
      trailer = sql.slice(openIndex);
    }
  }
  return (
    body
      .replace(/'(?:[^'\\]|\\.)*'/g, "?")
      .replace(/"(?:[^"\\]|\\.)*"/g, "?")
      // Decimal, hex (0x1a2b), and exponent (1e21) literal forms.
      .replace(/\b(?:0x[0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b/g, "?") + trailer
  );
}

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
  for (const key of SQL_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      attributes[key] = maskSqlLiterals(value);
    }
  }
}

/**
 * Matches a URL glued to a non-URL prefix inside a single token, e.g.
 * `fetch(/api/keys?key=x)` or `url=https://...`. Group 1 is the prefix up to
 * and including the `=` or `(`, group 2 the URL itself, and group 3 any
 * trailing `)` wrapper characters — kept out of the URL so `fetch(...)`
 * retains its balanced paren instead of the `)` being percent-encoded into
 * the last query value. `/(?!\*)` keeps comment openers (`=/*`) from being
 * mistaken for a path.
 */
const EMBEDDED_URL = /^(.*?[=(])((?:https?:\/\/|\/(?!\*)).*?)(\)*)$/i;

/**
 * Scrubs URLs embedded in free-form text such as span descriptions
 * (`GET https://.../keys?token=x`, `GET /api/keys?key=x [200]`) and
 * transaction names (`/auth/callback`, `middleware GET /api/keys`). Works
 * token-by-token across any whitespace so text around the URL survives, and
 * text without a URL-shaped token (db/ui span descriptions,
 * `trpc/key.create`-style names) is returned unchanged. A URL glued to a
 * prefix such as `url=` or `fetch(` is still found and scrubbed.
 */
function scrubUrlsInText(text: string): string {
  return text
    .split(/(\s+)/)
    .map((token) => {
      if (token.length === 0 || /^\s/.test(token)) {
        return token;
      }
      // SQLCommenter blocks (`/*service='...',route='...'*/` on db span
      // descriptions) start with a slash but are not URLs; rewriting them
      // corrupts the Query Insights tags and merges distinct queries.
      if (token.startsWith("/*")) {
        return token;
      }
      if (token.startsWith("/") || /^https?:\/\//i.test(token)) {
        return scrubUrl(token);
      }
      const embedded = token.match(EMBEDDED_URL);
      if (embedded) {
        return embedded[1] + scrubUrl(embedded[2]) + embedded[3];
      }
      return token;
    })
    .join("");
}

/**
 * Whether a span records an executed SQL statement. Sentry's OpenTelemetry
 * layer copies `db.statement` verbatim into the span description, so db spans
 * need SQL masking on the description too, not just on the attribute.
 */
function isSqlSpan(span: { op?: string; data?: Record<string, unknown> }): boolean {
  if (span.op?.startsWith("db")) {
    return true;
  }
  return SQL_ATTRIBUTE_KEYS.some((key) => typeof span.data?.[key] === "string");
}

/**
 * Scrubs the URL-bearing fields shared by transaction child spans and
 * standalone span envelopes — the free-text description plus the span
 * attributes — in place. Both envelope paths call this single helper so the
 * scrubbed surfaces cannot drift apart. Db span descriptions are the raw SQL
 * statement (copied from `db.statement`), so they get literal masking rather
 * than URL scrubbing.
 */
function scrubSpanFields(span: {
  op?: string;
  description?: string;
  data?: Record<string, unknown>;
}): void {
  if (typeof span.description === "string") {
    span.description = isSqlSpan(span)
      ? maskSqlLiterals(span.description)
      : scrubUrlsInText(span.description);
  }
  scrubSpanAttributes(span.data);
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
      // When a db span is the trace root (a query running outside any request
      // span), the transaction name IS the raw interpolated SQL statement.
      event.transaction = isSqlSpan({
        op: event.contexts?.trace?.op,
        data: event.contexts?.trace?.data,
      })
        ? maskSqlLiterals(event.transaction)
        : scrubUrlsInText(event.transaction);
    }
    scrubRequest(event.request);
    scrubBreadcrumbs(event);

    // The tRPC middleware attaches raw procedure input to `contexts.trpc`,
    // which merges into transaction events too — the same leak the error path
    // closes via `createErrorFilter`.
    scrubTrpcInput(event);

    // The root span's attributes live on the trace context, not in `spans`.
    scrubSpanAttributes(event.contexts?.trace?.data);

    for (const span of event.spans ?? []) {
      scrubSpanFields(span);
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
 * their page URL rides in the `transaction` attribute. The hook also runs on
 * each child span of a transaction, which `scrubTransactionPii` scrubs again;
 * that double pass is deliberate, so the transaction path's fail-closed
 * guarantee never depends on this hook being registered.
 *
 * Scrubbing works on a shallow copy — built in its own guarded step so even a
 * span whose property access throws cannot make the hook throw into the SDK —
 * and `beforeSendSpan` offers no drop path (the SDK requires a span back), so
 * if scrubbing fails the copy is returned with its scrubbed-surface fields
 * removed rather than forwarded unscrubbed.
 */
export function scrubSpanPii(span: SpanJson): SpanJson {
  let copy: SpanJson;
  try {
    copy = { ...span, data: span.data ? { ...span.data } : span.data };
  } catch {
    // Even reading the span's properties threw (an exotic proxy). There is
    // nothing safer to build, so return the original rather than throw into
    // envelope processing.
    return span;
  }

  try {
    scrubSpanFields(copy);
    return copy;
  } catch {
    delete copy.description;
    if (copy.data) {
      for (const key of SCRUBBED_ATTRIBUTE_KEYS) {
        delete copy.data[key];
      }
    }
    return copy;
  }
}
