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

  it("drops the fragment entirely (it can carry the /share bearer id)", () => {
    expect(scrubUrl("/share#ss_abc123")).toBe("/share");
    expect(scrubUrl("https://app.unkey.com/share#ss_abc123")).toBe("https://app.unkey.com/share");
  });

  it("never throws on malformed input", () => {
    expect(() => scrubUrl("http://[::1::bad")).not.toThrow();
    expect(scrubUrl("")).toBe("");
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

    expect(returned).toBe(span);
    expect(JSON.stringify(span.data)).not.toContain(ROOT_KEY);
    // Non-URL text like the interaction target selector is untouched.
    expect(span.description).toBe("body > div#root > button.submit");
    expect(span.data["sentry.exclusive_time"]).toBe(120);
  });
});
