import { REDACTED, redactTokenLike, scrubUrl } from "@/lib/sentry";
import * as Sentry from "@sentry/nextjs";
import { after } from "next/server";

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
 * refresh-token rotation. Everything below treats the body as hostile:
 * only allowlisted, correctly-typed fields are forwarded, body size is capped
 * in bytes while streaming, and forwarding is rate limited so this endpoint
 * cannot be used to burn the Sentry project's shared event quota.
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
const FORWARD_LIMIT_PER_WINDOW = 60;
const FORWARD_WINDOW_MS = 60_000;

// Sliding log of forward timestamps (at most FORWARD_LIMIT_PER_WINDOW
// entries): any 60s span forwards at most the limit. A fixed reset-to-zero
// window would let a burst straddling the boundary forward double.
const forwardTimestamps: number[] = [];

// Give up on a slow Sentry ingest quickly; forwarding is best-effort.
const FORWARD_TIMEOUT_MS = 3_000;

/**
 * The report-uri fields we forward, per the CSP2 violation-report format.
 * Unknown fields and wrong-typed values are dropped, never copied through:
 * this keeps the endpoint from relaying arbitrary attacker JSON to Sentry.
 * `script-sample` is deliberately absent — it can contain page content.
 */

/** The string fields that can carry URLs and must be scrubbed. */
const URL_FIELDS = new Set(["document-uri", "referrer", "blocked-uri", "source-file"]);

// Derived from the two parts so URL_FIELDS ⊂ STRING_FIELDS holds
// structurally — a URL field added to only one set cannot silently become
// dead scrubbing config.
const NON_URL_STRING_FIELDS = [
  "violated-directive",
  "effective-directive",
  "original-policy",
  "disposition",
];
const STRING_FIELDS = new Set([...NON_URL_STRING_FIELDS, ...URL_FIELDS]);

const NUMBER_FIELDS = new Set(["status-code", "line-number", "column-number"]);

// Per-field cap on forwarded strings. The longest legitimate value is
// original-policy (our own policy string, ~1KB); anything bigger is attacker
// padding, not a real report field.
const MAX_STRING_FIELD_CHARS = 2_048;

/**
 * Fallback for values scrubUrl cannot handle (non-http(s) schemes,
 * unparseable input): everything from the first "?" or "#" is dropped
 * wholesale — these values never reach scrubUrl's name-aware param
 * scrubbing, and short named secrets (password=, email=) evade the 20+ char
 * token net. The rest gets token-shape redaction, which leaves short CSP
 * keywords untouched.
 */
function redactOpaqueValue(value: string): string {
  const cutAt = value.search(/[?#]/);
  const cut = cutAt === -1 ? value : `${value.slice(0, cutAt)}${value[cutAt]}${REDACTED}`;
  return redactTokenLike(cut);
}

/**
 * blocked-uri/document-uri are not always http(s) URLs: browsers send
 * keywords ("inline", "eval", "self", "data", "about:blank") as well as
 * other schemes (blob:, wss:). scrubUrl would parse keywords against its
 * dummy base and mangle them into fake paths ("/inline"), so it only runs on
 * values that are parseable http(s) URLs; everything else gets the wholesale
 * redaction above. The parseability probe matters: scrubUrl's own
 * parse-failure fallback is token-shape only, which short named secrets
 * evade.
 */
function scrubUrlField(value: string): string {
  const trimmed = value.trim();
  if (!/^https?:\/\//i.test(trimmed)) {
    return redactOpaqueValue(trimmed);
  }
  try {
    new URL(trimmed);
  } catch {
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

/**
 * Single source of truth for the limit check. `consume: false` is the cheap
 * pre-check that sheds floods before any body streaming/parsing; a token is
 * only consumed (`consume: true`) once a report has validated.
 */
function checkForwardBudget(now: number, options: { consume: boolean }): boolean {
  for (;;) {
    const oldest = forwardTimestamps[0];
    if (oldest === undefined || now - oldest < FORWARD_WINDOW_MS) {
      break;
    }
    forwardTimestamps.shift();
  }
  if (forwardTimestamps.length >= FORWARD_LIMIT_PER_WINDOW) {
    return false;
  }
  if (options.consume) {
    forwardTimestamps.push(now);
  }
  return true;
}

export async function POST(request: Request): Promise<Response> {
  const dsn = Sentry.getClient()?.getDsn();
  if (!dsn) {
    return new Response(null, { status: 204 });
  }

  // Shed over-limit floods before streaming/parsing the body. The check does
  // not consume a token, so invalid bodies never burn forward budget.
  if (!checkForwardBudget(Date.now(), { consume: false })) {
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

  const scrubbed: Record<string, string | number> = {};
  for (const [key, value] of Object.entries(report)) {
    if (STRING_FIELDS.has(key) && typeof value === "string") {
      // Scrub before truncating so the redaction pass always sees the full
      // value; the cap then bounds what attacker padding can push to Sentry.
      const forwarded = URL_FIELDS.has(key) ? scrubUrlField(value) : value;
      scrubbed[key] = forwarded.slice(0, MAX_STRING_FIELD_CHARS);
    } else if (NUMBER_FIELDS.has(key) && typeof value === "number" && Number.isFinite(value)) {
      // isFinite: Infinity/NaN would pass a bare typeof check but serialize
      // to null, forwarding a wrong-typed value despite the allowlist.
      scrubbed[key] = value;
    }
  }

  // A report with no recognized field is not a real browser report; reject
  // it before it can consume forward budget or reach Sentry as a contentless
  // event.
  if (Object.keys(scrubbed).length === 0) {
    return new Response(null, { status: 400 });
  }

  if (!checkForwardBudget(Date.now(), { consume: true })) {
    return new Response(null, { status: 204 });
  }

  const port = dsn.port ? `:${dsn.port}` : "";
  const path = dsn.path ? `/${dsn.path}` : "";
  const endpoint = `${dsn.protocol}://${dsn.host}${port}${path}/api/${dsn.projectId}/security/?sentry_key=${dsn.publicKey}`;

  // Respond immediately; the Sentry round-trip runs after the response so a
  // slow ingest never holds the browser's report request open.
  after(async () => {
    try {
      await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/csp-report" },
        body: JSON.stringify({ "csp-report": scrubbed }),
        signal: AbortSignal.timeout(FORWARD_TIMEOUT_MS),
      });
    } catch {
      // Reporting is best-effort; swallow forwarding failures (including the
      // timeout abort).
    }
  });

  return new Response(null, { status: 204 });
}
