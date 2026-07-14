import { redactOpaqueValue, redactTokenLike, scrubUrl } from "@/lib/sentry";
import * as Sentry from "@sentry/nextjs";
import { after } from "next/server";

/** The DSN parts the Sentry SDK exposes; the security endpoint is built from them. */
type SentryDsn = NonNullable<
  ReturnType<NonNullable<ReturnType<typeof Sentry.getClient>>["getDsn"]>
>;

/**
 * Receives browser CSP violation reports (`report-uri /api/csp-reports` in
 * next.config.js) and forwards them to Sentry's security endpoint.
 *
 * Reports never go to Sentry directly from the browser because their
 * document-uri/blocked-uri carry raw page URLs — including bearer credentials
 * like `?invitation_token=` — that must pass through the same scrubbing as
 * every other Sentry payload (lib/sentry/pii-scrubber.ts). Routing them here
 * also inherits the SDK's environment gating: when Sentry is not initialized
 * (development, SENTRY_DISABLED, self-hosted opt-out) there is no client and
 * reports are dropped instead of phoning home to Unkey's Sentry project, and
 * the ingest endpoint is derived from the SDK's DSN instead of duplicating it
 * here.
 *
 * Unauthenticated by design: violations on pre-auth pages (sign-in, invite
 * links) must be reportable, and the path is in proxy.ts publicPaths so the
 * browser's credentialed report POSTs never enter authMiddleware's
 * refresh-token rotation. Everything below treats the request as hostile:
 * it must be shaped like a browser report (Content-Type, Fetch Metadata),
 * only allowlisted, correctly-typed fields are forwarded and scrubbed, body
 * size is capped in bytes while streaming, and forwarding is rate limited
 * both globally and per distinct violation so this endpoint cannot be used to
 * burn the Sentry project's shared event quota and no single noisy violation
 * can crowd out the rest.
 */

// Byte cap on the report body. Real browser reports are well under 4KB; the
// stream reader below enforces this in actual bytes before buffering more.
const MAX_REPORT_BYTES = 16_384;

// Forwarding shares the Sentry project's quota with real error events and the
// endpoint is unauthenticated, so cap forwards per instance and window.
// Reports beyond the cap are still answered 204 (reporting is
// fire-and-forget) but dropped.
//
// Deliberately per-instance: on serverless every instance gets its own
// window, so a concurrent flood can multiply past this cap in aggregate.
// Bounding that would require shared limiter state (external store), which
// is not warranted for best-effort telemetry — Sentry's own spike protection
// and per-key rate limits are the aggregate backstop; this cap bounds what a
// single instance will relay.
//
// A deliberate flood of well-formed forged reports with varied signatures can
// still exhaust this budget — an accepted limitation of an unauthenticated
// endpoint, since per-client attribution would again need shared state. The
// request-shape checks in POST bound that to deliberate direct forgery rather
// than trivial curl loops or cross-site browser floods, and drops are
// surfaced (see reportDropped) instead of vanishing.
const FORWARD_LIMIT_PER_WINDOW = 60;
const FORWARD_WINDOW_MS = 60_000;

// Per-distinct-violation cap, well under the global one. Without it a single
// noisy page — one violation repeated on every pageview — spends the whole
// global budget, and the rare violation that would actually break users on
// enforcement is dropped: the one report this whole feature exists to catch.
// Forwarding a violation five times per minute is plenty, because Sentry
// groups identical security reports anyway; the sixth duplicate carries no
// information the first five did not.
const FORWARD_LIMIT_PER_SIGNATURE = 5;

// Sliding logs of forward timestamps: any 60s span forwards at most the
// limit. A fixed reset-to-zero window would let a burst straddling the
// boundary forward double.
const forwardTimestamps: number[] = [];

// Keyed by violation signature. Only *forwarded* reports are recorded, so the
// map holds at most FORWARD_LIMIT_PER_WINDOW live signatures and emptied
// entries are deleted — an attacker cannot grow it by varying the signature.
const signatureTimestamps = new Map<string, number[]>();

// Give up on a slow Sentry ingest quickly; forwarding is best-effort.
const FORWARD_TIMEOUT_MS = 3_000;

// The allowlist of report-uri fields we forward, per the CSP2
// violation-report format. Everything else — unknown fields, wrong-typed
// values — is dropped rather than copied through, so the endpoint cannot relay
// arbitrary attacker JSON to Sentry. `script-sample` is deliberately absent:
// it can contain page content.

/** The fields that can carry URLs, and so must go through URL scrubbing. */
const URL_FIELDS = new Set(["document-uri", "referrer", "blocked-uri", "source-file"]);

const NON_URL_STRING_FIELDS = [
  "violated-directive",
  "effective-directive",
  "original-policy",
  "disposition",
];

// Derived from the two parts so URL_FIELDS ⊂ STRING_FIELDS holds
// structurally — a URL field added to only one set cannot silently become
// dead scrubbing config.
const STRING_FIELDS = new Set([...NON_URL_STRING_FIELDS, ...URL_FIELDS]);

const NUMBER_FIELDS = new Set(["status-code", "line-number", "column-number"]);

// Per-field cap on forwarded strings. The longest legitimate value is
// original-policy (our own policy string, ~1KB); anything bigger is attacker
// padding, not a real report field.
const MAX_STRING_FIELD_CHARS = 2_048;

// The media types browsers use for `report-uri`: Chrome and Firefox send
// application/csp-report, WebKit sends application/json. Both carry the same
// `{"csp-report": {...}}` body, which is the only shape parsed below.
//
// application/reports+json is deliberately absent. It belongs to the Reporting
// API (`report-to`), which the policy in next.config.js does not use, and its
// body is an array of `{type, body}` envelopes rather than a csp-report
// object — accepting the media type without teaching the parser that shape
// would advertise support that does not exist. Adding `report-to` means
// changing both.
//
// Neither accepted type is CORS-simple, so a cross-site fetch() carrying one
// is preflighted, and this route answers no preflight. That is what makes the
// check load-bearing against browser-driven floods: a no-cors flood can only
// set the simple types (x-www-form-urlencoded, multipart/form-data,
// text/plain), which this rejects.
const REPORT_CONTENT_TYPE = /^application\/(csp-report|json)\s*(;|$)/i;

/**
 * blocked-uri/document-uri are not always http(s) URLs: browsers send
 * keywords ("inline", "eval", "self", "data", "about:blank") as well as
 * other schemes (blob:, wss:). scrubUrl would parse keywords against its
 * dummy base and mangle them into fake paths ("/inline"), so it only runs on
 * values that look like http(s) URLs; everything else gets
 * redactOpaqueValue's wholesale query/fragment/userinfo drop, which is also
 * scrubUrl's own fallback for http(s) values that fail URL parsing.
 */
function scrubUrlField(value: string): string {
  const trimmed = value.trim();
  if (!/^https?:\/\//i.test(trimmed)) {
    return redactOpaqueValue(trimmed);
  }
  return scrubUrl(trimmed);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * Reads the request body while counting actual bytes, so the cap holds for
 * multi-byte payloads (string length counts UTF-16 code units, undercounting
 * UTF-8 bytes up to 4x) and oversized bodies stop buffering at the cap
 * instead of after being read in full. Returns null when the cap is exceeded.
 */
async function readBodyWithByteCap(request: Request, maxBytes: number): Promise<string | null> {
  const body = request.body;
  if (!body) {
    return "";
  }

  const reader = body.getReader();
  const chunks: Uint8Array[] = [];
  let received = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    received += value.byteLength;
    if (received > maxBytes) {
      await reader.cancel();
      return null;
    }
    chunks.push(value);
  }

  const combined = new Uint8Array(received);
  let offset = 0;
  for (const chunk of chunks) {
    combined.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(combined);
}

/** Drops timestamps that have aged out of the sliding window, in place. */
function pruneExpired(timestamps: number[], now: number): void {
  for (;;) {
    const oldest = timestamps[0];
    if (oldest === undefined || now - oldest < FORWARD_WINDOW_MS) {
      return;
    }
    timestamps.shift();
  }
}

/**
 * Cheap pre-check that sheds floods before any body streaming or parsing. It
 * consumes nothing, so invalid bodies never burn forward budget; the
 * per-signature budget cannot be checked yet because the signature is only
 * known once the report has parsed and scrubbed.
 */
function hasGlobalForwardBudget(now: number): boolean {
  pruneExpired(forwardTimestamps, now);
  return forwardTimestamps.length < FORWARD_LIMIT_PER_WINDOW;
}

/**
 * Groups reports by what makes two of them the *same* violation: the directive
 * that fired and the origin that was blocked. The path is deliberately not
 * part of the key — a page loading twenty blocked scripts from one host is one
 * violation to fix, not twenty. Built from already-scrubbed values so no raw
 * report data reaches the limiter's keys.
 */
function violationSignature(report: Record<string, string | number>): string {
  const directive = report["effective-directive"] ?? report["violated-directive"] ?? "";
  const blocked = String(report["blocked-uri"] ?? "");
  // Keywords ("inline", "eval") have no origin to extract; keep them whole.
  const origin = blocked.match(/^[a-z][a-z0-9+.-]*:\/\/[^/?#]*/i)?.[0] ?? blocked;
  return `${String(directive)}|${origin}`;
}

/**
 * Deletes signatures whose forwards have all aged out. Bounds the map to the
 * signatures actually forwarded in the current window.
 */
function pruneSignatures(now: number): void {
  for (const [signature, timestamps] of signatureTimestamps) {
    pruneExpired(timestamps, now);
    if (timestamps.length === 0) {
      signatureTimestamps.delete(signature);
    }
  }
}

/**
 * Claims one forward token for a validated report. Both budgets must have
 * room: the global one bounds what this instance relays, the per-signature one
 * keeps a single repeated violation from spending it all.
 *
 * Must stay synchronous. Concurrent requests on this instance interleave only
 * at await points, so checking a budget and pushing the timestamp is atomic
 * exactly as long as nothing awaits between them. Introducing an await here
 * would let two in-flight reports both observe room and both claim it,
 * overshooting the cap.
 */
function claimForwardBudget(now: number, signature: string): boolean {
  if (!hasGlobalForwardBudget(now)) {
    return false;
  }

  const forwards = signatureTimestamps.get(signature) ?? [];
  pruneExpired(forwards, now);
  if (forwards.length >= FORWARD_LIMIT_PER_SIGNATURE) {
    return false;
  }

  forwards.push(now);
  signatureTimestamps.set(signature, forwards);
  forwardTimestamps.push(now);
  pruneSignatures(now);
  return true;
}

// A dropped report is invisible: the browser gets the same 204 either way, so
// a saturated limiter looks exactly like a clean policy — and "the report
// stream is quiet" is the signal used to decide whether to promote the
// report-only policy to enforced. Surface saturation instead of letting it
// masquerade as success. Per-signature drops are deliberate deduplication, not
// lost signal, so only global exhaustion is reported here.
let droppedSinceLastReport = 0;
let lastDropReportAtMs = 0;

/**
 * Reports global-budget drops to Sentry at most once per window, carrying the
 * count. Rate limited because the alarm must not burn the very quota the
 * forward budget exists to protect.
 */
function reportDropped(now: number): void {
  droppedSinceLastReport++;
  if (lastDropReportAtMs !== 0 && now - lastDropReportAtMs < FORWARD_WINDOW_MS) {
    return;
  }

  const dropped = droppedSinceLastReport;
  droppedSinceLastReport = 0;
  lastDropReportAtMs = now;
  Sentry.captureMessage(
    `CSP report forwarding saturated: dropped ${dropped} report(s) in the last window`,
    "warning",
  );
}

/**
 * Copies the allowlisted fields of a hostile report into a forwardable one.
 * Unknown keys and wrong-typed values are dropped rather than copied through,
 * and every surviving string is scrubbed: this is the only thing standing
 * between attacker JSON and the Sentry security feed.
 */
function scrubReport(report: Record<string, unknown>): Record<string, string | number> {
  const scrubbed: Record<string, string | number> = {};

  for (const [key, value] of Object.entries(report)) {
    if (STRING_FIELDS.has(key) && typeof value === "string") {
      // Scrub before truncating so the redaction pass always sees the full
      // value; the cap then bounds what attacker padding can push to Sentry.
      //
      // Non-URL fields still get token-shape redaction rather than passing
      // through verbatim: they are attacker-controllable on this
      // unauthenticated endpoint, and original-policy echoes our own policy
      // string — which carries a per-request nonce once script-src is
      // nonce-ified (see next.config.js).
      const forwarded = URL_FIELDS.has(key) ? scrubUrlField(value) : redactTokenLike(value);
      scrubbed[key] = forwarded.slice(0, MAX_STRING_FIELD_CHARS);
    } else if (NUMBER_FIELDS.has(key) && typeof value === "number" && Number.isFinite(value)) {
      // isFinite: Infinity/NaN would pass a bare typeof check but serialize
      // to null, forwarding a wrong-typed value despite the allowlist.
      scrubbed[key] = value;
    }
  }

  return scrubbed;
}

/**
 * Relays a scrubbed report to the Sentry security endpoint derived from the
 * SDK's own DSN. Runs after the response so a slow ingest never holds the
 * browser's report request open, and swallows failures because reporting is
 * best-effort.
 */
function forwardToSentry(dsn: SentryDsn, scrubbed: Record<string, string | number>): void {
  const port = dsn.port ? `:${dsn.port}` : "";
  const path = dsn.path ? `/${dsn.path}` : "";
  const endpoint = `${dsn.protocol}://${dsn.host}${port}${path}/api/${dsn.projectId}/security/?sentry_key=${dsn.publicKey}`;

  after(async () => {
    try {
      await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/csp-report" },
        body: JSON.stringify({ "csp-report": scrubbed }),
        signal: AbortSignal.timeout(FORWARD_TIMEOUT_MS),
      });
    } catch {
      // Swallow forwarding failures, including the timeout abort.
    }
  });
}

export async function POST(request: Request): Promise<Response> {
  const dsn = Sentry.getClient()?.getDsn();
  if (!dsn) {
    return new Response(null, { status: 204 });
  }

  // Shape validation, not authentication — a direct client can forge these
  // headers. The point is to stop cross-site browser-driven floods and naive
  // abuse before any budget or body work, so both checks are deliberately
  // permissive about *real* browser behavior and strict only where a value
  // proves the request is not a same-origin report POST. A report wrongly
  // rejected here is invisible: it would look like a clean policy rather
  // than a suppressed one, which is exactly the mistake that gets a broken
  // CSP promoted from report-only to enforced.

  const contentType = request.headers.get("content-type")?.trim() ?? "";
  if (!REPORT_CONTENT_TYPE.test(contentType)) {
    return new Response(null, { status: 415 });
  }

  // report-uri points at this route on the page's own origin, so a genuine
  // report POST is never cross-site. Reject only that value: "same-origin",
  // "same-site" (violating document on a sibling subdomain) and "none" (no
  // document initiator) are all legitimate, and the header is absent
  // entirely in browsers without Fetch Metadata.
  if (request.headers.get("sec-fetch-site") === "cross-site") {
    return new Response(null, { status: 403 });
  }

  // Shed over-limit floods before streaming/parsing the body. Nothing is
  // consumed here, so invalid bodies never burn forward budget.
  const arrivedAtMs = Date.now();
  if (!hasGlobalForwardBudget(arrivedAtMs)) {
    reportDropped(arrivedAtMs);
    return new Response(null, { status: 204 });
  }

  // Fast-reject declared-oversized bodies; the stream cap below still holds
  // when the header is absent or lies.
  const contentLength = Number(request.headers.get("content-length"));
  if (Number.isFinite(contentLength) && contentLength > MAX_REPORT_BYTES) {
    return new Response(null, { status: 413 });
  }

  let report: Record<string, unknown>;
  try {
    const raw = await readBodyWithByteCap(request, MAX_REPORT_BYTES);
    if (raw === null) {
      return new Response(null, { status: 413 });
    }
    const parsed: unknown = JSON.parse(raw);
    // report-uri delivers `{"csp-report": {...}}` with Content-Type
    // application/csp-report.
    if (!isRecord(parsed) || !isRecord(parsed["csp-report"])) {
      return new Response(null, { status: 400 });
    }
    report = parsed["csp-report"];
  } catch {
    return new Response(null, { status: 400 });
  }

  const scrubbed = scrubReport(report);

  // A report with no recognized field is not a real browser report; reject
  // it before it can consume forward budget or reach Sentry as a contentless
  // event.
  if (Object.keys(scrubbed).length === 0) {
    return new Response(null, { status: 400 });
  }

  const claimedAtMs = Date.now();
  if (!claimForwardBudget(claimedAtMs, violationSignature(scrubbed))) {
    // Only global exhaustion is a lost signal worth alarming on; a
    // per-signature drop means this exact violation already reached Sentry
    // FORWARD_LIMIT_PER_SIGNATURE times this window.
    if (!hasGlobalForwardBudget(claimedAtMs)) {
      reportDropped(claimedAtMs);
    }
    return new Response(null, { status: 204 });
  }

  forwardToSentry(dsn, scrubbed);

  return new Response(null, { status: 204 });
}
