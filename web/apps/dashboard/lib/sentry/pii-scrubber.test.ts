import type { ErrorEvent } from "@sentry/nextjs";
import { describe, expect, it } from "vitest";
import {
  type SpanJson,
  type TransactionEvent,
  scrubEventPii,
  scrubSpanPii,
  scrubTransactionPii,
  scrubUrl,
} from "./pii-scrubber";

const ROOT_KEY = "unkey_3ZZ8gT8vQk2mN4pXwYbCdEf";
const JWT =
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N";

describe("scrubUrl", () => {
  it("redacts sensitive query params by name", () => {
    expect(scrubUrl("/api/verify?key=secret123&foo=bar")).toBe(
      "/api/verify?key=%5BREDACTED%5D&foo=bar",
    );
    expect(scrubUrl("https://app.unkey.com/auth?code=abc&state=xyz")).toContain(
      "code=%5BREDACTED%5D",
    );
  });

  it("redacts token-like values even under unknown param names", () => {
    const out = scrubUrl(`/x?ref=${ROOT_KEY}`);
    expect(out).not.toContain(ROOT_KEY);
    // Query param values are URL-encoded, so the marker appears as %5BREDACTED%5D.
    expect(out).toContain("REDACTED");
  });

  it("redacts digit-free 20+ char values under unknown param names", () => {
    // Query values are not route names, so the broad net applies: passphrase-style
    // letters-only secrets must not reach Sentry just because they lack a digit.
    const out = scrubUrl("/verify?ref=abcdefghijklmnopqrstuvwx");
    expect(out).not.toContain("abcdefghijklmnopqrstuvwx");
  });

  it("preserves repeated query params while scrubbing", () => {
    // `URLSearchParams.set` during iteration collapses duplicates, which would
    // silently corrupt multi-value filters in the captured URL.
    expect(scrubUrl("/keys?status=active&status=trialing&key=secret123")).toBe(
      "/keys?status=active&status=trialing&key=%5BREDACTED%5D",
    );
  });

  it("redacts opaque-path scheme payloads wholesale", () => {
    // The WHATWG pathname setter is a no-op on opaque paths, so mailto:/tel:/
    // blob: payloads would otherwise pass through the path machinery
    // untouched — and their payload is inherently identifying.
    expect(scrubUrl("mailto:jane.doe@example.com")).toBe("mailto:[REDACTED]");
    expect(scrubUrl("tel:+15551234567")).toBe("tel:[REDACTED]");
    expect(scrubUrl("blob:https://app.unkey.com/1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")).toBe(
      "blob:[REDACTED]",
    );
  });

  it("redacts ClickHouse param_ query bindings regardless of the bound name", () => {
    // @clickhouse/client sends bound query params as `param_<name>`; bindings
    // are data values (external ids are often emails) whose short runs evade
    // the token net, so the name prefix alone must trigger redaction.
    const out = scrubUrl(
      "https://ch.unkey.com/?param_externalId=jane.doe%40acme.com&database=unkey",
    );
    expect(out).not.toContain("jane.doe");
    expect(out).toContain("database=unkey");
  });

  it("drops basic-auth userinfo entirely", () => {
    const out = scrubUrl("https://dbuser:secretpass@clickhouse.example.com/query?x=1");
    expect(out).not.toContain("secretpass");
    expect(out).not.toContain("dbuser");
    expect(out).toContain("clickhouse.example.com/query");
  });

  it("redacts token-like segments embedded in the path", () => {
    const out = scrubUrl(`/keys/${ROOT_KEY}/details`);
    expect(out).not.toContain(ROOT_KEY);
    expect(out.startsWith("/keys/")).toBe(true);
  });

  it("redacts JWTs", () => {
    expect(scrubUrl(`/cb?token=${JWT}`)).not.toContain(JWT);
  });

  it("preserves relative form and short non-sensitive values", () => {
    expect(scrubUrl("/projects/abc/apps?page=2")).toBe("/projects/abc/apps?page=2");
  });

  it("fully redacts tRPC GET input payloads (short PII evades the token net)", () => {
    const out = scrubUrl('/api/trpc/key.list?batch=1&input={"json":{"email":"jo@acme.com"}}');
    expect(out).not.toContain("jo%40acme.com");
    expect(out).not.toContain("jo@acme.com");
    expect(out).toContain("batch=1");
  });

  it("does not redact long letter-only identifiers such as tRPC procedure names", () => {
    expect(scrubUrl("/api/trpc/ratelimit.queryRatelimitLatencyTimeseries?batch=1")).toBe(
      "/api/trpc/ratelimit.queryRatelimitLatencyTimeseries?batch=1",
    );
  });

  it("leaves Next.js build asset paths intact so resource spans stay distinguishable", () => {
    expect(scrubUrl("/_next/static/chunks/4407-8b62c8f2a3d1e5f6.js")).toBe(
      "/_next/static/chunks/4407-8b62c8f2a3d1e5f6.js",
    );
  });

  it("still redacts token-like segments under /_next/data (only static assets are exempt)", () => {
    // `/_next/data/` payload URLs embed dynamic route params, which can be
    // bearer-style ids — the exemption must not cover them.
    const out = scrubUrl(`/_next/data/build-id/${ROOT_KEY}.json`);
    expect(out).not.toContain(ROOT_KEY);
  });

  it("drops the fragment entirely (it can carry the /share bearer id)", () => {
    expect(scrubUrl("/share#ss_abc123")).toBe("/share");
    expect(scrubUrl("https://app.unkey.com/share#ss_abc123")).toBe("https://app.unkey.com/share");
  });

  it("never throws on malformed input and still redacts tokens in it", () => {
    expect(() => scrubUrl("http://[::1::bad")).not.toThrow();
    expect(scrubUrl("")).toBe("");
    // The parse-failure fallback must stay fail-closed: token-like runs are
    // blanket-redacted even when the URL has no parseable structure.
    expect(scrubUrl(`http://[::1::bad/${ROOT_KEY}`)).not.toContain(ROOT_KEY);
  });
});

describe("scrubEventPii", () => {
  it("scrubs request url, query string, and breadcrumb urls in place", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: {
        url: `https://app.unkey.com/keys?key=${ROOT_KEY}`,
        query_string: `key=${ROOT_KEY}&page=1`,
      },
      breadcrumbs: [
        {
          category: "fetch",
          data: { url: `https://api.unkey.com/v1/keys.verifyKey?token=${JWT}` },
        },
        {
          category: "navigation",
          data: { from: `/login?code=${ROOT_KEY}`, to: "/dashboard" },
        },
      ],
    };

    scrubEventPii(event);

    expect(event.request?.url).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.request?.query_string)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).not.toContain(JWT);
    // Non-sensitive breadcrumb data is preserved.
    expect(JSON.stringify(event.breadcrumbs)).toContain("/dashboard");
  });

  it("is a no-op on events without request or breadcrumbs", () => {
    const event: ErrorEvent = { type: undefined };
    expect(() => scrubEventPii(event)).not.toThrow();
  });

  it("scrubs tRPC input even when URL scrubbing throws afterwards", () => {
    // The credential surface must be redacted BEFORE the fail-open URL
    // scrubbing: a throw there sends the event as-is, so running the input
    // scrub last would ship raw credentials exactly when scrubbing breaks.
    const event: ErrorEvent = {
      type: undefined,
      breadcrumbs: [
        {
          category: "navigation",
          data: Object.freeze({ from: `/login?code=${ROOT_KEY}`, to: "/x" }),
        },
      ],
      contexts: {
        trpc: {
          procedure_path: "key.create",
          input: { secret: "unkey_secret_plaintext_value" },
        },
      },
    };

    expect(() => scrubEventPii(event)).not.toThrow();
    expect(event.contexts?.trpc?.input).toEqual({ secret: "[REDACTED]" });
    // The frozen breadcrumb really did make URL scrubbing throw (fail-open:
    // the event ships as-is) — otherwise this test would not be exercising
    // the ordering guarantee at all.
    expect(event.breadcrumbs?.[0]?.data?.from).toContain(ROOT_KEY);
  });

  it("drops the whole tRPC input when scrubbing it throws", () => {
    // Fail-closed guarantee for the credential-bearing surface: a scrub
    // failure must never forward the raw input (nor take down the error
    // report, as an unguarded throw in `beforeSend` previously would).
    const event: ErrorEvent = {
      type: undefined,
      contexts: {
        trpc: {
          procedure_path: "key.create",
          input: {
            get boom(): string {
              throw new Error("getter exploded");
            },
          },
        },
      },
    };

    scrubEventPii(event);

    expect(event.contexts?.trpc?.input).toBe("[REDACTED]");
  });
});

/**
 * Builds a minimal but fully typed span, so span fixtures stay checked against
 * the real Sentry span shape instead of being cast into it.
 */
function makeSpan(overrides: Partial<SpanJson>): SpanJson {
  return {
    span_id: "aaaaaaaaaaaaaaaa",
    trace_id: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    start_timestamp: 0,
    data: {},
    ...overrides,
  };
}

describe("scrubTransactionPii", () => {
  it("scrubs request url, query string, and breadcrumbs in place", () => {
    const event: TransactionEvent = {
      type: "transaction",
      transaction: "GET /auth/callback",
      request: {
        url: `https://app.unkey.com/auth/callback?code=${ROOT_KEY}`,
        query_string: `code=${ROOT_KEY}&state=xyz`,
      },
      breadcrumbs: [
        {
          category: "fetch",
          data: { url: `https://api.unkey.com/v1/keys.verifyKey?token=${JWT}` },
        },
      ],
    };

    const returned = scrubTransactionPii(event);

    expect(returned).toBe(event);
    expect(event.request?.url).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.request?.query_string)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).not.toContain(JWT);
  });

  it("scrubs URL-bearing headers and redacts credential headers", () => {
    const event: TransactionEvent = {
      type: "transaction",
      request: {
        headers: {
          Referer: `https://app.unkey.com/auth/callback?code=${ROOT_KEY}`,
          Authorization: `Bearer ${JWT}`,
          "user-agent": "Mozilla/5.0",
        },
      },
    };

    scrubTransactionPii(event);

    expect(event.request?.headers?.Referer).not.toContain(ROOT_KEY);
    expect(event.request?.headers?.Referer).toContain("/auth/callback");
    expect(event.request?.headers?.Authorization).toBe("[REDACTED]");
    expect(event.request?.headers?.["user-agent"]).toBe("Mozilla/5.0");
  });

  it("scrubs URL-carrying span attributes and descriptions", () => {
    const event: TransactionEvent = {
      type: "transaction",
      transaction: "/dashboard",
      spans: [
        makeSpan({
          op: "http.client",
          description: `GET https://api.unkey.com/v1/keys?key=${ROOT_KEY}`,
          data: {
            "http.url": `https://api.unkey.com/v1/keys?key=${ROOT_KEY}`,
            "url.full": `https://api.unkey.com/v1/keys?key=${ROOT_KEY}`,
            "http.query": `?key=${ROOT_KEY}`,
            "http.fragment": "ss_abc123",
            "http.method": "GET",
          },
        }),
        makeSpan({
          op: "http.client",
          description: `GET /api/internal?token=${JWT}`,
          data: { url: `/api/internal?token=${JWT}` },
        }),
      ],
    };

    scrubTransactionPii(event);

    const serialized = JSON.stringify(event.spans);
    expect(serialized).not.toContain(ROOT_KEY);
    expect(serialized).not.toContain(JWT);
    expect(serialized).not.toContain("ss_abc123");
    // Non-URL attributes and the URL structure itself survive.
    expect(event.spans?.[0]?.data["http.method"]).toBe("GET");
    expect(event.spans?.[0]?.data["http.url"]).toContain("https://api.unkey.com/v1/keys");
    // Query strings keep their leading `?` so the attribute format is stable.
    expect(event.spans?.[0]?.data["http.query"]).toMatch(/^\?key=/);
    expect(event.spans?.[1]?.description).toMatch(/^GET \/api\/internal\?token=/);
  });

  it("scrubs relative URLs embedded in prefixed names and suffixed descriptions", () => {
    const event: TransactionEvent = {
      type: "transaction",
      transaction: `middleware GET /auth/callback?code=${ROOT_KEY}`,
      spans: [
        makeSpan({
          op: "http.client",
          description: `GET /api/keys?key=${ROOT_KEY} [200]`,
        }),
      ],
    };

    scrubTransactionPii(event);

    expect(event.transaction).not.toContain(ROOT_KEY);
    expect(event.transaction).toContain("middleware GET /auth/callback");
    expect(event.spans?.[0]?.description).not.toContain(ROOT_KEY);
    expect(event.spans?.[0]?.description).toContain("[200]");
  });

  it("scrubs URLs glued to prefixes or separated by non-space whitespace", () => {
    const event: TransactionEvent = {
      type: "transaction",
      transaction: `GET\t/auth/callback?code=${ROOT_KEY}`,
      spans: [
        makeSpan({ description: `fetch(/api/keys?key=${ROOT_KEY})` }),
        makeSpan({ description: `redirect url=/login?token=${JWT}` }),
      ],
    };

    scrubTransactionPii(event);

    expect(event.transaction).not.toContain(ROOT_KEY);
    expect(event.transaction).toContain("GET\t/auth/callback");
    // Exact output: the `fetch(...)` wrapper keeps its balanced paren instead
    // of the `)` being percent-encoded into the last query value.
    expect(event.spans?.[0]?.description).toBe("fetch(/api/keys?key=%5BREDACTED%5D)");
    expect(event.spans?.[1]?.description).toBe("redirect url=/login?token=%5BREDACTED%5D");
  });

  it("leaves SQLCommenter blocks in db span descriptions untouched", () => {
    // The dashboard attaches `/*service='...',route='...'*/` comments to SQL via
    // createCommentedPool; the comment token starts with a slash but is not a
    // URL, and rewriting it would corrupt Query Insights tags and merge
    // distinct queries in Sentry Performance.
    const description =
      "SELECT * FROM `keys` WHERE id = ? /*service='dashboard',route='trpc/queryRatelimitLatency2Timeseries'*/";
    const event: TransactionEvent = {
      type: "transaction",
      spans: [makeSpan({ op: "db", description })],
    };

    scrubTransactionPii(event);

    expect(event.spans?.[0]?.description).toBe(description);
  });

  it("scrubs the root span attributes on the trace context", () => {
    const event: TransactionEvent = {
      type: "transaction",
      contexts: {
        trace: {
          span_id: "aaaaaaaaaaaaaaaa",
          trace_id: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
          data: { "url.full": `https://app.unkey.com/auth/callback?code=${ROOT_KEY}` },
        },
      },
    };

    scrubTransactionPii(event);

    expect(JSON.stringify(event.contexts)).not.toContain(ROOT_KEY);
  });

  it("scrubs secrets from transaction names without mangling route names", () => {
    const withSecret: TransactionEvent = {
      type: "transaction",
      transaction: `/auth/callback?code=${ROOT_KEY}`,
    };
    scrubTransactionPii(withSecret);
    expect(withSecret.transaction).not.toContain(ROOT_KEY);

    for (const name of [
      "GET /api/keys",
      "/settings/root-keys/[keyId]",
      "trpc/key.create",
      "/api/trpc/ratelimit.queryRatelimitLatencyTimeseries",
    ]) {
      const event: TransactionEvent = { type: "transaction", transaction: name };
      scrubTransactionPii(event);
      expect(event.transaction).toBe(name);
    }
  });

  it("masks SQL literals in transaction names when a db span is the trace root", () => {
    // A query running outside any request span becomes its own transaction,
    // named with the raw interpolated statement — URL scrubbing alone would
    // leave the literals intact.
    const statement = "SELECT * FROM `keys` WHERE owner_email = 'jo@acme.com'";
    const event: TransactionEvent = {
      type: "transaction",
      transaction: statement,
      contexts: {
        trace: {
          span_id: "aaaaaaaaaaaaaaaa",
          trace_id: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
          op: "db",
          data: { "db.statement": statement },
        },
      },
    };

    scrubTransactionPii(event);

    expect(event.transaction).toBe("SELECT * FROM `keys` WHERE owner_email = ?");
    expect(JSON.stringify(event.contexts)).not.toContain("jo@acme.com");
  });

  it("scrubs sensitive fields from tRPC input attached to transaction contexts", () => {
    // `attachRpcInput: true` writes raw procedure input to the scope, and
    // scope contexts merge into transaction events too — without this,
    // 100%-sampled tRPC transactions would carry what `beforeSend` redacts on
    // error events.
    const event: TransactionEvent = {
      type: "transaction",
      transaction: "trpc/key.create",
      contexts: {
        trpc: {
          procedure_path: "key.create",
          input: { secret: "unkey_secret_plaintext_value", safe: "kept" },
        },
      },
    };

    scrubTransactionPii(event);

    expect(event.contexts?.trpc?.input).toEqual({ secret: "[REDACTED]", safe: "kept" });
  });

  it("fully redacts credential-procedure input on transactions", () => {
    // A *successful* share.reveal samples at 100% as a tRPC transaction; its
    // input id is the one-time bearer credential and must never reach Sentry.
    const shareId = "still_valid_one_time_share_id";
    const event: TransactionEvent = {
      type: "transaction",
      transaction: "trpc/share.reveal",
      contexts: {
        trpc: { procedure_path: "share.reveal", input: { id: shareId } },
      },
    };

    scrubTransactionPii(event);

    expect(event.contexts?.trpc?.input).toBe("[REDACTED]");
    expect(JSON.stringify(event)).not.toContain(shareId);
  });

  it("drops the transaction instead of sending it half-scrubbed when scrubbing throws", () => {
    const frozenSpan = makeSpan({ description: `GET /x?key=${ROOT_KEY}` });
    Object.freeze(frozenSpan);
    const event: TransactionEvent = { type: "transaction", spans: [frozenSpan] };

    expect(scrubTransactionPii(event)).toBeNull();
  });

  it("is a no-op on transactions without request, spans, or contexts", () => {
    const event: TransactionEvent = { type: "transaction" };
    expect(scrubTransactionPii(event)).toBe(event);
  });
});

describe("scrubSpanPii", () => {
  it("scrubs the transaction attribute standalone web-vital spans carry", () => {
    const span = makeSpan({
      op: "ui.interaction.click",
      description: "body > div#root > button.submit",
      data: {
        transaction: `/share/${ROOT_KEY}`,
        "sentry.exclusive_time": 120,
      },
    });

    const returned = scrubSpanPii(span);

    expect(JSON.stringify(returned.data)).not.toContain(ROOT_KEY);
    // Non-URL text like the interaction target selector is untouched.
    expect(returned.description).toBe("body > div#root > button.submit");
    expect(returned.data["sentry.exclusive_time"]).toBe(120);
  });

  it("masks SQL literal values in db statement attributes while keeping query shape", () => {
    // The mysql2 auto-instrumentation interpolates bound values into
    // `db.query.text`; literals must be masked, but identifiers and the
    // SQLCommenter tags must survive for query grouping and Query Insights.
    const span = makeSpan({
      op: "db",
      data: {
        "db.query.text":
          "SELECT * FROM `keys` WHERE owner_email = 'jo@acme.com' AND id = 42 /*service='dashboard'*/",
      },
    });

    const returned = scrubSpanPii(span);

    expect(returned.data["db.query.text"]).toBe(
      "SELECT * FROM `keys` WHERE owner_email = ? AND id = ? /*service='dashboard'*/",
    );
  });

  it("masks SQL literals in db span descriptions too (Sentry copies db.statement there)", () => {
    // Sentry's OpenTelemetry layer sets a db span's description to the raw
    // `db.statement` verbatim — masking only the attribute would leave the
    // displayed field leaking the same literals.
    const statement = "SELECT * FROM `keys` WHERE owner_email = 'jo@acme.com' AND id = 42";
    const span = makeSpan({
      op: "db",
      description: statement,
      data: { "db.statement": statement },
    });

    const returned = scrubSpanPii(span);

    expect(returned.description).toBe("SELECT * FROM `keys` WHERE owner_email = ? AND id = ?");
    expect(returned.data["db.statement"]).toBe(
      "SELECT * FROM `keys` WHERE owner_email = ? AND id = ?",
    );
  });

  it("masks hex and exponent numeric literal forms", () => {
    const span = makeSpan({
      op: "db",
      description: "SELECT * FROM t WHERE flags = 0x1A2B AND score > 1e21",
    });

    const returned = scrubSpanPii(span);

    expect(returned.description).toBe("SELECT * FROM t WHERE flags = ? AND score > ?");
  });

  it("masks literals even when a value contains a comment opener", () => {
    // A `/*` inside a user-supplied string must not fake a comment opener and
    // smuggle the surrounding literals through unmasked up to the SQLCommenter
    // trailer.
    const span = makeSpan({
      op: "db",
      description:
        "SELECT * FROM keys WHERE note = 'contact a/*b jo@acme.com' AND id = 42 /*service='dashboard'*/",
    });

    const returned = scrubSpanPii(span);

    expect(returned.description).toBe(
      "SELECT * FROM keys WHERE note = ? AND id = ? /*service='dashboard'*/",
    );
  });

  it("scrubs read-only spans by copying instead of forwarding them unscrubbed", () => {
    // `beforeSendSpan` has no drop path, so a span that cannot be mutated in
    // place must still come back scrubbed — via a copy, not as-is.
    const span = makeSpan({ data: { transaction: `/share/${ROOT_KEY}` } });
    Object.freeze(span.data);
    Object.freeze(span);

    const returned = scrubSpanPii(span);

    expect(JSON.stringify(returned)).not.toContain(ROOT_KEY);
  });
});
