/**
 * Sentry PII Scrubbing
 *
 * Centralized scrubbing of personally identifiable information and secrets from
 * Sentry payloads.
 *
 * `sendDefaultPii: false` buys far less than its name suggests: in the SDK's
 * `RequestData` integration only the client IP (and the IP-bearing headers) is
 * gated on it, while `cookies`, `headers`, `data`, `query_string` and `url` are
 * all included by default and attached to server/edge events regardless. It
 * also never scrubs data that already lives inside values we generate
 * ourselves. Root keys, JWTs, OAuth codes and similar secrets routinely appear
 * in query strings and path segments, and those URLs surface in:
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
 * Secrets also reach Sentry through the payload's free-form surfaces. That
 * option gates *whether* a field is collected, never *what* it contains, so it
 * offers these nothing either:
 *
 *   - error messages → `event.exception.values[].value`, `event.message`,
 *                      `event.logentry` — a thrown error that interpolates a
 *                      root key, a URL or an env-var value into its message
 *                      (CWE-209) would otherwise be forwarded verbatim
 *   - attached data  → `event.extra`, `event.request.data`
 *
 * Messages get `scrubText` (URL scrubbing plus the digit-bearing token net);
 * the structured surfaces get the same key-based redaction as tRPC input.
 *
 * Stack frames are deliberately left alone: Sentry matches source maps on
 * `filename`/`abs_path`, and the JS SDK does not collect frame-local variables
 * (the `localVariables` integration is not enabled in any of our configs).
 *
 * `contexts` is scrubbed by name rather than wholesale. `contexts.trpc` holds
 * procedure input and is redacted; `contexts.trace` ids are token-like, so a
 * blanket pass would sever trace linking. The rest is either the SDK's own
 * (browser, os, device) or app-attached and known non-sensitive — today only
 * `react.componentStack` from `error-boundary.tsx`, which is component names.
 * A new `setContext` carrying user data needs its own case here.
 *
 * Note the SDK normalizes `extra`, `contexts` and `breadcrumbs.data` (cutting
 * cycles, resolving exotic objects) *before* `beforeSend` runs, so those
 * arrive here as plain JSON. `logentry` is not on that list — see
 * `scrubMessage`.
 *
 * This module redacts those values before the payload leaves the browser/server.
 * Error events run through `beforeSend`, transactions through
 * `beforeSendTransaction`, standalone spans through `beforeSendSpan`; the hooks
 * fire on disjoint payloads, so each family needs its own wiring in the Sentry
 * configs.
 */

import type { ErrorEvent, EventHint, NodeOptions, replayIntegration } from "@sentry/nextjs";
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
 * redacts every match; `redactDigitBearingTokens` additionally requires a digit
 * before redacting.
 */
const TOKEN_LIKE = /[A-Za-z0-9_-]{20,}/g;

/**
 * Matches an email address. `TOKEN_LIKE` cannot catch these — its alphabet
 * stops at `.` and `@`, so `john.doe@customer.com` has no run over 20 chars —
 * yet `email` is in `SENSITIVE_NAME_KEYS`, i.e. we already classify addresses
 * as PII wherever we can see the field name. Where there is no field name to
 * read (prose, a `?q=` search value, a log attribute) the shape is the only
 * signal left. Deliberately not extended to phone numbers: digit runs are far
 * too easy to confuse with versions, counts and ids.
 */
const EMAIL_LIKE = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;

/**
 * Whether a parameter/field name is known to carry secrets or PII. Matched
 * case-insensitively. Shared by the URL param scrubber and the tRPC input
 * scrubber so both surfaces treat the same names as sensitive.
 */
function isSensitiveKey(name: string): boolean {
  return SENSITIVE_NAME_KEYS.has(name.toLowerCase());
}

/**
 * Redacts token-like substrings and emails from a value regardless of its field
 * name, as a fail-closed fallback for secrets and PII under unrecognized names.
 * Applied everywhere route grouping is not a concern — query param values, tRPC
 * input field values, log attributes, unparseable URLs — so digit-free secrets
 * (passphrases, letters-only ids) fail closed.
 *
 * Emails go first: redacting tokens first could consume a long local part and
 * leave `[REDACTED]@customer.com`, whose `]` breaks the email match and strands
 * the domain.
 */
function redactTokenLike(value: string): string {
  return value.replace(EMAIL_LIKE, REDACTED).replace(TOKEN_LIKE, REDACTED);
}

/**
 * Redacts token-like runs only when the run contains a digit. Opaque secrets
 * (Unkey root keys, JWTs, session ids) mix digits into their alphabet; the
 * digit requirement spares long human-written identifiers like tRPC procedure
 * names (`getDeploymentRuntimeLogs`), which would otherwise collapse to
 * [REDACTED] and merge distinct routes/issues in Sentry. Used on the two
 * surfaces where that grouping matters and text is human-authored — URL paths
 * and free-form messages — accepting a known residual miss: a digit-free 20+
 * char secret. Everywhere else `redactTokenLike` casts the broader net.
 *
 * The digit check runs per matched run rather than as a regex lookahead; a
 * lookahead re-scans from every position, turning long digit-free input into
 * an ~O(n²) stall on the browser main thread.
 */
function redactDigitBearingTokens(text: string): string {
  return text.replace(TOKEN_LIKE, (run) => (/\d/.test(run) ? REDACTED : run));
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
 * schemes (mailto:, tel:, blob:, data:) are redacted wholesale down to the
 * scheme. Any other opaque-path URL (`urn:`, `sms:`, an app deep-link) keeps
 * its scheme and gets path-level token redaction rather than being dropped.
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
  // number, a blob id). Redact the payload wholesale, keeping the scheme. The
  // allowlist is explicit so unrelated colon-prefixed strings that reach here
  // (a Windows path like `C:\Users\...`, a `word:` breadcrumb) fall through to
  // normal parsing instead of being mangled into `word:[REDACTED]`.
  const opaqueScheme = url.match(/^(mailto|tel|blob|data):(?!\/\/)/i)?.[1];
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

    // Opaque-path URLs outside the wholesale-redaction allowlist above (an
    // unrecognized `scheme:opaque` such as `urn:`, `sms:`, or an app
    // deep-link) parse with a scheme but an empty host, and the WHATWG
    // `pathname` setter is a no-op on them. Reconstruct `scheme:path` manually
    // so the scheme is preserved and the opaque path still gets token
    // redaction, instead of falling through to the relative branch below,
    // which would strip the scheme and skip path scrubbing entirely.
    const hadScheme = /^[a-z][a-z0-9+.-]*:/i.test(url);
    if (hadScheme && parsed.host === "" && !url.startsWith("//")) {
      const scrubbedPath = redactDigitBearingTokens(parsed.pathname);
      return `${parsed.protocol}${scrubbedPath}${parsed.search}`;
    }

    // Redact token-like segments embedded directly in the path. Next.js static
    // build assets are exempt: their hashed chunk names look token-like but
    // are public, immutable files, and redacting them would make every
    // resource span indistinguishable in Sentry Performance. Only
    // `/_next/static/` is exempt — `/_next/data/` payload URLs embed dynamic
    // route params, which can be token-like ids.
    if (!parsed.pathname.startsWith("/_next/static/")) {
      parsed.pathname = redactDigitBearingTokens(parsed.pathname);
    }

    // Drop the fragment entirely. It is never useful for debugging and can carry
    // bearer credentials, e.g. the one-time share id in `/share#<id>` links.
    parsed.hash = "";

    // Protocol-relative URLs (`//host/path`) parse with a real host under the
    // dummy base but carry no scheme, so reconstruct them as `//host/...` to
    // keep the host for span grouping instead of dropping it.
    if (url.startsWith("//")) {
      return `//${parsed.host}${parsed.pathname}${parsed.search}`;
    }

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
 * Scrubs the URLs and free-form text carried by breadcrumbs. Sentry's default
 * fetch/xhr/navigation breadcrumbs put URLs under `data.url`, `data.from` and
 * `data.to`, while the console integration puts the formatted console
 * arguments in `breadcrumb.message` — so `console.error("...", rootKey)`
 * (which `error-boundary.tsx` does with the caught error) would otherwise
 * carry the secret verbatim into the next event, right past the scrubbing
 * `exception.values[].value` gets.
 */
function scrubBreadcrumbs(event: ErrorEvent | TransactionEvent, run: StepRunner): void {
  if (!event.breadcrumbs) {
    return;
  }
  // Per-breadcrumb rather than per-loop: one exotic breadcrumb must not abort
  // the iteration and leave every breadcrumb after it unscrubbed.
  for (const breadcrumb of event.breadcrumbs) {
    run(() => scrubBreadcrumb(breadcrumb));
  }
}

/**
 * Scrubs one breadcrumb in place.
 */
function scrubBreadcrumb(breadcrumb: NonNullable<ErrorEvent["breadcrumbs"]>[number]): void {
  // `ui.click`/`ui.input` messages are `htmlTreeAsString` selector paths, not
  // prose: their ids and hashed class names are exactly the digit-bearing
  // token shape `scrubText` redacts, which would reduce the click trail to
  // `div > [REDACTED] > button`. Every other category's message is free text
  // (console output, custom `addBreadcrumb`) and gets scrubbed.
  if (typeof breadcrumb.message === "string" && !breadcrumb.category?.startsWith("ui.")) {
    breadcrumb.message = scrubText(breadcrumb.message);
  }
  const data = breadcrumb.data;
  if (!data) {
    return;
  }
  for (const field of ["url", "from", "to"] as const) {
    const value = data[field];
    if (typeof value === "string") {
      data[field] = scrubUrl(value);
    }
  }
}

/**
 * Scrubs PII/secrets from an error event in place: the attached tRPC input,
 * `extra`, exception and log messages, request URLs/headers/body, and
 * breadcrumbs. Safe to call on every event regardless of classification. Never
 * throws.
 *
 * @param event - The Sentry error event to scrub. Mutated in place because
 *   Sentry consumes the same object returned from `beforeSend`.
 */
export function scrubEventPii(event: ErrorEvent, _hint?: EventHint): void {
  // Every pass is isolated: a throw in one must never skip the others. A
  // single try/catch around the whole sequence would let a failure in an
  // early pass silently forward the URLs a later pass exists to redact.
  // Within a pass the trade-off differs — the surfaces holding plaintext
  // secrets under our own control (tRPC input, `extra`) fail closed inside
  // their own helpers and drop the payload, while the rest fail open, because
  // losing the whole error report costs more than one unscrubbed field.
  isolate(() => scrubTrpcInput(event));
  isolate(() => scrubExtra(event));
  isolate(() => scrubExceptions(event));
  isolate(() => scrubMessage(event));
  scrubRequest(event.request, isolate);
  scrubBreadcrumbs(event, isolate);
}

/**
 * How a helper's individual steps are run. The two event families need
 * opposite failure behavior from the *same* scrubbing code, so the helpers
 * that both paths share take the runner rather than hard-coding a `try`:
 * error events fail open and pass `isolate`, so one bad step cannot skip the
 * rest; transactions fail closed and pass `propagate`, so any throw reaches
 * `scrubTransactionPii`'s catch and drops the whole payload.
 */
type StepRunner = (step: () => void) => void;

/**
 * Runs one scrubbing step, swallowing any throw. Scrubbing must never prevent
 * an error from being reported, and must never let one failing surface stop
 * the remaining surfaces from being scrubbed.
 */
function isolate(step: () => void): void {
  try {
    step();
  } catch {
    // Deliberately empty: see `scrubEventPii`.
  }
}

/**
 * Runs one scrubbing step with no isolation, letting a throw reach a
 * fail-closed caller that will drop the payload outright.
 */
function propagate(step: () => void): void {
  step();
}

/**
 * Matches a `key=value&key=value` form-encoded body. Anchored and deliberately
 * strict about the key: `URLSearchParams` happily "parses" any string at all
 * (`hello world` becomes the key `hello world`), so without a shape check a
 * plain-text body would be mangled into `hello+world=`.
 */
const FORM_ENCODED = /^[\w.%+-]+=[^&]*(?:&[\w.%+-]+=[^&]*)*$/;

/**
 * Scrubs an `event.request.data` body. Bodies usually arrive as a raw *string*,
 * where key-based redaction would never fire on its own — walking keys needs a
 * structure, and the token net cannot see that the `hunter2` in
 * `{"password":"hunter2"}` is a password. So the two encodings we actually
 * receive are decoded first: JSON is parsed, redacted by key and re-serialized;
 * form-encoded bodies reuse the query-string scrubber, which is already
 * key-aware. A body of any other shape (plain text) has no field names to read
 * and falls back to the blind token net.
 */
function scrubRequestData(data: unknown): unknown {
  if (typeof data !== "string") {
    return redactSensitiveValues(data);
  }

  // Only `JSON.parse` belongs in the `try`. Wrapping the redaction too would
  // catch a genuine scrubbing failure and quietly downgrade it to the token
  // net over the *raw* string, forwarding `{"password":"hunter2"}` intact —
  // the opposite of the caller's fail-closed drop, which a rethrow reaches.
  let parsed: unknown;
  let isJson = false;
  try {
    parsed = JSON.parse(data);
    isJson = true;
  } catch {
    // Not JSON; try the other encodings below.
  }

  if (isJson && parsed && typeof parsed === "object") {
    return JSON.stringify(redactSensitiveValues(parsed));
  }
  if (FORM_ENCODED.test(data)) {
    return scrubQueryParamsString(data);
  }
  return redactTokenLike(data);
}

/**
 * Headers whose values are URLs and can carry secrets in their query strings,
 * e.g. the OAuth code in a `Referer` sent by a page loaded as
 * `/auth/callback?code=...`.
 */
const URL_HEADERS = new Set(["referer", "referrer", "location"]);

/**
 * Headers whose values are credentials and are redacted wholesale. Contrary to
 * what the option's name suggests, `sendDefaultPii: false` does not drop the
 * `cookie` header — the `RequestData` integration includes `cookies` and
 * `headers` by default and gates only the IP on that option — so on
 * server/edge events every one of these arrives verbatim unless redacted here.
 */
const SENSITIVE_HEADERS = new Set([
  "authorization",
  "proxy-authorization",
  "cookie",
  "set-cookie",
  "x-api-key",
]);

/**
 * Scrubs the URL, query string, body, cookies, and URL/credential-bearing
 * headers of an event's `request` payload in place.
 *
 * Each surface is its own step: they are independent, and the credential-
 * bearing ones (cookies, headers) sit last, so on the fail-open error path a
 * throw while scrubbing a URL must not skip them. See `StepRunner`.
 */
function scrubRequest(
  request: ErrorEvent["request"] | TransactionEvent["request"],
  run: StepRunner,
): void {
  if (!request) {
    return;
  }

  run(() => {
    if (typeof request.url === "string") {
      request.url = scrubUrl(request.url);
    }
  });

  run(() => {
    if (request.query_string != null) {
      request.query_string = scrubQueryString(request.query_string);
    }
  });

  // Request bodies are attached by the SDK itself on server/edge — the
  // `RequestData` integration includes `data` by default and does NOT gate it
  // on `sendDefaultPii` — so this runs on real captured bodies, not just the
  // ones our code sets explicitly. A body is the same shape of structured data
  // as tRPC input, so it gets the same treatment, failing closed to a drop
  // rather than forwarding raw.
  run(() => {
    if (request.data == null) {
      return;
    }
    try {
      request.data = scrubRequestData(request.data);
    } catch {
      delete request.data;
    }
  });

  // The same integration parses the `cookie` header into `request.cookies`,
  // also un-gated by `sendDefaultPii`. Redacting only the header would leave
  // the session token sitting in plain sight one field over — every cookie
  // value is a credential or a tracking id, so all of them go. Names are kept:
  // knowing *which* cookies were present is useful and is not the secret.
  run(() => {
    if (!request.cookies) {
      return;
    }
    for (const name of Object.keys(request.cookies)) {
      request.cookies[name] = REDACTED;
    }
  });

  run(() => {
    if (!request.headers) {
      return;
    }
    for (const [name, value] of Object.entries(request.headers)) {
      const lowerName = name.toLowerCase();
      if (SENSITIVE_HEADERS.has(lowerName)) {
        request.headers[name] = REDACTED;
      } else if (URL_HEADERS.has(lowerName) && typeof value === "string") {
        request.headers[name] = scrubUrl(value);
      }
    }
  });
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
 * Recursively replaces the values of sensitive keys with a redaction marker,
 * walking nested objects and arrays so secrets under `variables`/`items` are
 * caught too, and scrubbing the remaining leaf strings with `scrubString` as a
 * fail-closed fallback for secrets under names the key list doesn't anticipate.
 * Returns a new structure and never mutates the input.
 *
 * Callers pass the string policy: structured surfaces that carry no grouping
 * (`extra`, tRPC input, request bodies) take the broad `redactTokenLike` net,
 * while log/console values take `scrubText` so diagnostic constants survive.
 *
 * `ancestors` holds the current path and breaks reference cycles: some surfaces
 * (`logentry.params`, replay console args, request bodies) reach us
 * un-normalized, so a parent-linked object would otherwise recurse until the
 * stack blows and take the whole pass down. Each entry is released on the way
 * out, because it is a *path*, not a seen-set: the second occurrence of a
 * merely repeated reference (`{ a: shared, b: shared }`) is not a cycle and
 * must not be flagged. Only an object that contains itself is.
 *
 * Exotic objects (`Date`, `Error`, class instances) are passed through
 * untouched rather than rebuilt from `Object.entries`, which would flatten
 * their non-enumerable state to `{}`. The trade-off is that a secret in an
 * exotic object's own fields is a residual miss; in practice these surfaces
 * hold plain JSON or scalars.
 */
function redactStructured(
  value: unknown,
  scrubString: (s: string) => string,
  ancestors: WeakSet<object>,
): unknown {
  if (typeof value === "string") {
    return scrubString(value);
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  if (ancestors.has(value)) {
    return REDACTED;
  }

  const prototype = Object.getPrototypeOf(value);
  if (!Array.isArray(value) && prototype !== Object.prototype && prototype !== null) {
    return value;
  }

  ancestors.add(value);
  try {
    if (Array.isArray(value)) {
      return value.map((item) => redactStructured(item, scrubString, ancestors));
    }
    return redactStructuredRecord(value as Record<string, unknown>, scrubString, ancestors);
  } finally {
    ancestors.delete(value);
  }
}

/**
 * The plain-object case of `redactStructured`, split out so callers holding a
 * known record (`event.extra`, `log.attributes`) get a record back and need no
 * cast. Returns a new object and never mutates the input.
 */
function redactStructuredRecord(
  record: Record<string, unknown>,
  scrubString: (s: string) => string,
  ancestors: WeakSet<object>,
): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, nested] of Object.entries(record)) {
    result[key] = isSensitiveInputKey(key)
      ? REDACTED
      : redactStructured(nested, scrubString, ancestors);
  }
  return result;
}

/**
 * Structured redaction with the broad token/email net, for surfaces that carry
 * no issue grouping: tRPC input, `extra`, request bodies, replay console args.
 */
function redactSensitiveValues(value: unknown): unknown {
  return redactStructured(value, redactTokenLike, new WeakSet());
}

function redactSensitiveRecord(record: Record<string, unknown>): Record<string, unknown> {
  return redactStructuredRecord(record, redactTokenLike, new WeakSet());
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
 * Prefixes under which Sentry's Node HTTP instrumentation writes request and
 * response headers onto span/trace `data`, as `http.request.header.<name>` /
 * `http.response.header.<name>` (name lowercased, dashes as underscores). Its
 * own sensitivity filter redacts auth/token/cookie headers but NOT
 * `referer`/`referrer`/`location`, so a `Referer` carrying an OAuth code on a
 * 100%-sampled `/auth/callback` transaction reaches Sentry verbatim unless we
 * URL-scrub these here. The transaction's root span becomes the trace, so the
 * attribute surfaces on `event.contexts.trace.data`.
 */
const HEADER_ATTRIBUTE_PREFIXES = ["http.request.header.", "http.response.header."];

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
  // URL-bearing request/response headers (Referer/Location) carried as
  // `http.{request,response}.header.<name>` attributes. The instrumentation
  // lowercases the header name, so match against the lowercased `URL_HEADERS`
  // suffix. Iterated over the live keys because the header name is dynamic and
  // not enumerable ahead of time.
  for (const [key, value] of Object.entries(attributes)) {
    if (typeof value !== "string") {
      continue;
    }
    const prefix = HEADER_ATTRIBUTE_PREFIXES.find((p) => key.startsWith(p));
    if (prefix && URL_HEADERS.has(key.slice(prefix.length))) {
      attributes[key] = scrubUrl(value);
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
  return mapTextTokens(text, (token) => token);
}

/**
 * Splits free-form text on whitespace and routes each token to exactly one
 * treatment: URL-shaped tokens to `scrubUrl`, everything else to
 * `scrubNonUrlToken`. Applying only one pass per token is the point — layering
 * a second blanket pass over the joined result would re-redact what `scrubUrl`
 * deliberately preserved, e.g. the `/_next/static/` chunk hashes it exempts on
 * purpose.
 */
function mapTextTokens(text: string, scrubNonUrlToken: (token: string) => string): string {
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
      return scrubNonUrlToken(token);
    })
    .join("");
}

/**
 * Scrubs secrets from free-form prose: exception values and messages.
 *
 * Emails are matched by shape and redacted FIRST, across the whole string,
 * because they span characters the token alphabet excludes. Order matters: an
 * address inside a URL query (`/users?q=jane@acme.com`) would otherwise reach
 * `scrubUrl`, come back percent-encoded as `q=jane%40acme.com`, and no longer
 * contain an `@` for the matcher to find.
 *
 * Each whitespace token then gets exactly one treatment — URL-shaped tokens are
 * scrubbed structurally by `scrubUrl` (so a root key in a query string goes
 * even though it is short), and every other token gets
 * `redactDigitBearingTokens`, which nets opaque secrets interpolated as bare
 * words (`Error: key unkey_3ZjK...`) without collapsing the human-written
 * identifiers Sentry groups issues by.
 *
 * The residual miss is a short secret under a name only the prose reveals
 * (`password=hunter2`). Name-based redaction is deliberately not applied here:
 * `code`, `state` and `auth` are in `SENSITIVE_NAME_KEYS` and appear
 * constantly in benign message text (`code: NOT_FOUND`), so keying off them
 * would shred messages and merge unrelated issues for a case the structured
 * surfaces (`extra`, tRPC input) already cover.
 */
function scrubText(text: string): string {
  return mapTextTokens(text.replace(EMAIL_LIKE, REDACTED), redactDigitBearingTokens);
}

/**
 * Scrubs the exception messages of an error event in place. `values[].value`
 * is the thrown error's message; `values[].type` is its constructor name (a
 * code-level identifier, never data) and is left alone, as are stack frames —
 * see the module header.
 */
function scrubExceptions(event: ErrorEvent): void {
  for (const exception of event.exception?.values ?? []) {
    if (typeof exception.value === "string") {
      exception.value = scrubText(exception.value);
    }
  }
}

/**
 * Scrubs an event's message surfaces in place: the top-level `message` set by
 * `captureMessage`, plus the `logentry` form that `fmt` tagged templates
 * produce (`message` holding a `%s` template, `params` the values Sentry
 * interpolates into it server-side).
 *
 * `logentry` is the one surface the SDK's `normalizeEvent` skips, so unlike
 * `extra` it arrives with live values — cycles uncut, exotic objects
 * unresolved — which is why params go through `redactLogValue` (cycle-safe,
 * exotic-preserving) rather than the plain broad net.
 */
function scrubMessage(event: ErrorEvent): void {
  if (typeof event.message === "string") {
    event.message = scrubText(event.message);
  }

  const logentry = event.logentry;
  if (!logentry) {
    return;
  }
  // A template's literal text is developer-authored source, so it holds no
  // runtime secret — every runtime value sits in `params`. Scrubbing it would
  // only corrupt it: `scrubUrl` round-trips through `URLSearchParams`, which
  // re-encodes a `%s` placeholder to `%25s` and breaks interpolation. Scrub
  // the message only when it stands alone as prose.
  const hasParams = Array.isArray(logentry.params) && logentry.params.length > 0;
  if (typeof logentry.message === "string" && !hasParams) {
    logentry.message = scrubText(logentry.message);
  }
  if (Array.isArray(logentry.params)) {
    logentry.params = logentry.params.map(redactLogValue);
  }
}

/**
 * Structured redaction with the `scrubText` string policy (digit-bearing net,
 * emails, embedded URLs), for un-normalized diagnostic surfaces: `logentry`
 * params, `log.attributes`, replay console args. Unlike the broad net this
 * spares digit-free identifiers like `INTERNAL_SERVER_ERROR`, which these
 * surfaces log constantly and which are not secrets.
 */
function redactLogValue(value: unknown): unknown {
  return redactStructured(value, scrubText, new WeakSet());
}

function redactLogRecord(record: Record<string, unknown>): Record<string, unknown> {
  return redactStructuredRecord(record, scrubText, new WeakSet());
}

/**
 * Scrubs `event.extra` — the developer-attached bag from `setExtra`/
 * `captureException(e, { extra })` — in place, with the same key-based plus
 * broad token-like redaction as tRPC input: it is structured data no grouping
 * depends on, so it takes the wider net. Fails closed like the tRPC input
 * (a throwing getter drops the bag) and, because it runs before the URL
 * scrubbing that fails open, cannot be skipped by a later throw.
 */
function scrubExtra(event: ErrorEvent | TransactionEvent): void {
  if (!event.extra) {
    return;
  }
  try {
    event.extra = redactSensitiveRecord(event.extra);
  } catch {
    delete event.extra;
  }
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
    // `propagate`, not `isolate`: this path fails closed, so a throw must
    // reach the catch below and drop the transaction rather than send it
    // half-scrubbed.
    scrubRequest(event.request, propagate);
    scrubBreadcrumbs(event, propagate);

    // The tRPC middleware attaches raw procedure input to `contexts.trpc`,
    // which merges into transaction events too — the same leak the error path
    // closes via `createErrorFilter`. Scope `extra` merges into transactions
    // for the same reason.
    scrubTrpcInput(event);
    scrubExtra(event);

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
      // Header-carried URL attributes have dynamic keys the fixed list above
      // cannot enumerate, so prune them by prefix to keep the fallback aligned
      // with what `scrubSpanAttributes` touches.
      for (const key of Object.keys(copy.data)) {
        if (HEADER_ATTRIBUTE_PREFIXES.some((p) => key.startsWith(p))) {
          delete copy.data[key];
        }
      }
    }
    return copy;
  }
}

/**
 * The `beforeSendLog` payload. `@sentry/nextjs` does not re-export `Log`, so
 * derive it from the option this module's hook is wired into.
 */
export type SentryLog = Parameters<NonNullable<NodeOptions["beforeSendLog"]>>[0];

/**
 * Scrubs a Sentry structured log, for the `beforeSendLog` hook.
 *
 * Logs are a SEPARATE egress path: `enableLogs: true` is set in all three
 * configs and `Sentry.logger.*` envelopes never pass through `beforeSend`. That
 * matters because `createErrorFilter` deliberately drops expected tRPC errors
 * from the event pipeline and hands them to `logTRPCError` instead, which puts
 * the raw, pre-scrub `error.message` into the `trpc_error_message` attribute.
 * Without this hook, every secret `scrubExceptions` strips out of the event we
 * throw away would ride out in the log we keep.
 *
 * Both surfaces get the message policy — key-based redaction plus `scrubText`
 * on strings — rather than the broad net `extra` uses. A log attribute mostly
 * *is* a message (`trpc_error_message`) or a diagnostic constant, and the broad
 * net has no digit requirement, so it would redact `INTERNAL_SERVER_ERROR`
 * (21 chars of `[A-Z_]`) and throw away the very field being logged.
 * `redactLogRecord` also brings the cycle guard, which matters because
 * attributes are arbitrary caller-supplied values.
 *
 * Fails closed: `beforeSendLog` can drop, and a lost diagnostic log costs far
 * less than a leaked credential.
 */
export function scrubLog(log: SentryLog): SentryLog | null {
  try {
    if (typeof log.message === "string") {
      log.message = scrubText(log.message);
    }
    if (log.attributes) {
      log.attributes = redactLogRecord(log.attributes);
    }
    return log;
  } catch {
    return null;
  }
}

/**
 * The `beforeAddRecordingEvent` payload — a single Session Replay recording
 * frame. Derived from the replay integration's own option type.
 */
export type ReplayFrameEvent = Parameters<
  NonNullable<NonNullable<Parameters<typeof replayIntegration>[0]>["beforeAddRecordingEvent"]>
>[0];

/** The breadcrumb-tagged recording frame's payload (console, ui.*, custom). */
type ReplayBreadcrumbPayload = Extract<ReplayFrameEvent["data"], { tag: "breadcrumb" }>["payload"];
/** The span-tagged recording frame's payload (navigation, resource, fetch). */
type ReplaySpanPayload = Extract<ReplayFrameEvent["data"], { tag: "performanceSpan" }>["payload"];

/**
 * Recording-frame `data` keys whose values are URLs: fetch/xhr/navigation
 * breadcrumbs (`url`/`from`/`to`) and the history span's prior page
 * (`previous`).
 */
const REPLAY_URL_DATA_FIELDS = new Set(["url", "from", "to", "previous"]);

/**
 * Scrubs a Session Replay recording frame, for `beforeAddRecordingEvent`.
 *
 * Replay is the third egress path, and the leakiest: recording envelopes never
 * pass through `beforeSend`, and `replaysOnErrorSampleRate: 1.0` means every
 * error produces one. The `urls` array on the replay *event* is scrubbed by a
 * processor in the Sentry config, but the recording frames are a separate
 * envelope that only this hook sees.
 *
 * Two frame families carry secrets. Breadcrumb frames hold console output
 * (`error-boundary.tsx`'s `console.error("Tree layout error:", error, ...)`
 * would otherwise put the caught error's message into the replay verbatim) and
 * fetch/navigation URLs. Span frames (`performanceSpan`) put the resource or
 * navigation URL — query string and all — in `description`. Both are scrubbed;
 * option frames carry only config and are left alone.
 *
 * Fails closed by dropping the frame: replay frames are best-effort UI history,
 * and losing one is cheaper than shipping a root key.
 */
export function scrubReplayFrame(frame: ReplayFrameEvent): ReplayFrameEvent | null {
  try {
    const data = frame.data;
    if (data?.tag === "breadcrumb") {
      scrubReplayBreadcrumb(data.payload);
    } else if (data?.tag === "performanceSpan") {
      scrubReplaySpan(data.payload);
    }
    return frame;
  } catch {
    return null;
  }
}

/**
 * Scrubs a replay breadcrumb frame payload in place. Mirrors `scrubBreadcrumb`
 * on the event side — prose message (except `ui.*` selector paths) and URL data
 * fields — plus the console frame's raw `data.arguments`, which carry the same
 * values as the message. Console args are un-normalized live objects, so they
 * take `redactLogValue` (cycle-safe, exotic-preserving) with the same digit-net
 * string policy as the message, keeping one value from reading two ways within
 * one frame.
 */
function scrubReplayBreadcrumb(payload: ReplayBreadcrumbPayload): void {
  if (typeof payload.message === "string" && !payload.category?.startsWith("ui.")) {
    payload.message = scrubText(payload.message);
  }
  if (payload.category === "console" && payload.data && Array.isArray(payload.data.arguments)) {
    payload.data.arguments = payload.data.arguments.map(redactLogValue);
  }
  scrubReplayUrlDataFields(payload.data);
}

/**
 * Scrubs a replay span frame payload in place: the URL that resource/navigation
 * spans put in `description`, plus any URL-bearing `data` fields.
 */
function scrubReplaySpan(payload: ReplaySpanPayload): void {
  if (typeof payload.description === "string") {
    payload.description = scrubUrlsInText(payload.description);
  }
  scrubReplayUrlDataFields(payload.data);
}

/**
 * Scrubs the URL-valued keys of a recording frame's `data` in place. The frame
 * types are unions the SDK does not narrow for us, so this reads the object
 * structurally; only known URL keys with string values are rewritten.
 */
function scrubReplayUrlDataFields(data: unknown): void {
  if (!data || typeof data !== "object") {
    return;
  }
  const record = data as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    const value = record[key];
    if (typeof value === "string" && REPLAY_URL_DATA_FIELDS.has(key)) {
      record[key] = scrubUrl(value);
    }
  }
}
