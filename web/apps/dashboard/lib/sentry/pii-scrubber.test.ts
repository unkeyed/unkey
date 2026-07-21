import type { ErrorEvent } from "@sentry/nextjs";
import { describe, expect, it } from "vitest";
import {
  type ReplayFrameEvent,
  type SentryLog,
  type SpanJson,
  type TransactionEvent,
  scrubEventPii,
  scrubLog,
  scrubReplayFrame,
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
    expect(out).toContain("REDACTED");
  });

  it("redacts digit-free 20+ char values under unknown param names", () => {
    const out = scrubUrl("/verify?ref=abcdefghijklmnopqrstuvwx");
    expect(out).not.toContain("abcdefghijklmnopqrstuvwx");
  });

  it("preserves repeated query params while scrubbing", () => {
    expect(scrubUrl("/keys?status=active&status=trialing&key=secret123")).toBe(
      "/keys?status=active&status=trialing&key=%5BREDACTED%5D",
    );
  });

  it("redacts opaque-path scheme payloads wholesale", () => {
    expect(scrubUrl("mailto:jane.doe@example.com")).toBe("mailto:[REDACTED]");
    expect(scrubUrl("tel:+15551234567")).toBe("tel:[REDACTED]");
    expect(scrubUrl("blob:https://app.unkey.com/1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")).toBe(
      "blob:[REDACTED]",
    );
  });

  it("leaves non-opaque colon-prefixed strings unredacted (only known opaque schemes redact)", () => {
    expect(scrubUrl("git://github.com/unkeyed/unkey.git")).not.toContain("[REDACTED]");
    expect(scrubUrl("custom:hello/world")).toBe("custom:hello/world");
    expect(scrubUrl("custom:token12345678901234567890")).toBe("custom:[REDACTED]");
  });

  it("preserves the host for protocol-relative URLs", () => {
    expect(scrubUrl("//cdn.example.com/asset.js?x=1")).toBe("//cdn.example.com/asset.js?x=1");
  });

  it("redacts ClickHouse param_ query bindings regardless of the bound name", () => {
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
    expect(JSON.stringify(event.breadcrumbs)).toContain("/dashboard");
  });

  it("is a no-op on events without request or breadcrumbs", () => {
    const event: ErrorEvent = { type: undefined };
    expect(() => scrubEventPii(event)).not.toThrow();
  });

  it("scrubs tRPC input even when breadcrumb scrubbing throws", () => {
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
    expect(event.breadcrumbs?.[0]?.data?.from).toContain(ROOT_KEY);
  });

  it("keeps scrubbing later surfaces when an earlier pass throws", () => {
    const event: ErrorEvent = {
      type: undefined,
      exception: {
        values: [Object.freeze({ type: "Error", value: `boom ${ROOT_KEY}` })],
      },
      request: { url: `/keys?key=${ROOT_KEY}` },
      breadcrumbs: [{ category: "fetch", data: { url: `/v1/keys?token=${JWT}` } }],
    };

    scrubEventPii(event);

    expect(event.exception?.values?.[0]?.value).toContain(ROOT_KEY);
    expect(event.request?.url).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).not.toContain(JWT);
  });

  it("scrubs console breadcrumb messages", () => {
    const event: ErrorEvent = {
      type: undefined,
      breadcrumbs: [
        { category: "console", message: `failed to verify key ${ROOT_KEY}` },
        { category: "console", message: "no user found for john.doe@customer.com" },
      ],
    };

    scrubEventPii(event);

    expect(JSON.stringify(event.breadcrumbs)).not.toContain(ROOT_KEY);
    expect(event.breadcrumbs?.[1]?.message).toBe("no user found for [REDACTED]");
  });

  it("preserves _next/static chunk names in messages", () => {
    const chunk = "https://app.unkey.com/_next/static/chunks/app/layout-8f1a2b3c4d5e6f7a.js";
    const event: ErrorEvent = {
      type: undefined,
      exception: { values: [{ type: "ChunkLoadError", value: `Loading chunk failed (${chunk})` }] },
    };

    scrubEventPii(event);

    expect(event.exception?.values?.[0]?.value).toBe(`Loading chunk failed (${chunk})`);
  });

  it("scrubs string logentry params without destroying non-string ones", () => {
    const cyclic: Record<string, unknown> = {};
    cyclic.self = cyclic;
    const date = new Date(0);
    const event: ErrorEvent = {
      type: undefined,
      logentry: {
        message: "request to /search?q=%s failed",
        params: [`bare ${ROOT_KEY}`, date, cyclic],
      },
      request: { url: `/keys?key=${ROOT_KEY}` },
    };

    scrubEventPii(event);

    expect(event.logentry?.params?.[0]).toBe("bare [REDACTED]");
    expect(event.logentry?.params?.[1]).toBe(date);
    expect(event.logentry?.message).toBe("request to /search?q=%s failed");
    expect(event.request?.url).not.toContain(ROOT_KEY);
  });

  it("redacts by key inside a raw JSON string body", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: { data: '{"email":"a@b.com","password":"hunter2","keep":"ok"}' },
    };

    scrubEventPii(event);

    expect(event.request?.data).toBe('{"email":"[REDACTED]","password":"[REDACTED]","keep":"ok"}');
  });

  it("redacts by key inside a form-encoded string body", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: { data: "email=a%40b.com&password=hunter2&page=2" },
    };

    scrubEventPii(event);

    expect(event.request?.data).toContain("password=%5BREDACTED%5D");
    expect(event.request?.data).not.toContain("hunter2");
    expect(event.request?.data).toContain("page=2");
  });

  it("survives a cyclic object request body instead of dropping it", () => {
    const body: Record<string, unknown> = { password: "hunter2", keep: "ok" };
    body.self = body;
    const event: ErrorEvent = { type: undefined, request: { data: body } };

    scrubEventPii(event);

    const scrubbed = event.request?.data as Record<string, unknown>;
    expect(scrubbed.password).toBe("[REDACTED]");
    expect(scrubbed.keep).toBe("ok");
    expect(scrubbed.self).toBe("[REDACTED]");
  });

  it("leaves a plain-text body unmangled", () => {
    const event: ErrorEvent = { type: undefined, request: { data: "Something went wrong" } };

    scrubEventPii(event);

    expect(event.request?.data).toBe("Something went wrong");
  });

  it("redacts the parsed request cookies, not just the cookie header", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: {
        headers: { cookie: `__session=${JWT}` },
        cookies: { __session: JWT, theme: "dark" },
      },
    };

    scrubEventPii(event);

    expect(JSON.stringify(event.request?.cookies)).not.toContain(JWT);
    expect(event.request?.cookies?.__session).toBe("[REDACTED]");
    expect(Object.keys(event.request?.cookies ?? {})).toEqual(["__session", "theme"]);
  });

  it("still redacts credential headers when url scrubbing throws", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: {
        get url(): string {
          throw new Error("exotic getter");
        },
        headers: { authorization: `Bearer ${JWT}`, cookie: `__session=${JWT}` },
      },
    };

    scrubEventPii(event);

    expect(event.request?.headers?.authorization).toBe("[REDACTED]");
    expect(event.request?.headers?.cookie).toBe("[REDACTED]");
  });

  it("keeps scrubbing later breadcrumbs when one throws", () => {
    const event: ErrorEvent = {
      type: undefined,
      breadcrumbs: [
        Object.freeze({ category: "console", message: `boom ${ROOT_KEY}` }),
        { category: "fetch", data: { url: `/v1/keys?token=${JWT}` } },
      ],
    };

    scrubEventPii(event);

    expect(JSON.stringify(event.breadcrumbs?.[1])).not.toContain(JWT);
  });

  it("leaves ui.click selector paths intact", () => {
    const selector = "div > button.Button_root__1a2b3c4d5e6f7g8h > span";
    const event: ErrorEvent = {
      type: undefined,
      breadcrumbs: [{ category: "ui.click", message: selector }],
    };

    scrubEventPii(event);

    expect(event.breadcrumbs?.[0]?.message).toBe(selector);
  });

  it("redacts an email carried inside a URL query in prose", () => {
    const event: ErrorEvent = {
      type: undefined,
      exception: {
        values: [{ type: "Error", value: "Failed to fetch /api/users?q=john.doe@customer.com" }],
      },
    };

    scrubEventPii(event);

    const value = event.exception?.values?.[0]?.value;
    expect(value).not.toContain("john.doe");
    expect(value).not.toContain("customer.com");
  });

  it("scrubs a repeated param reference instead of treating it as a cycle", () => {
    const shared = { token: ROOT_KEY, keep: "visible" };
    const event: ErrorEvent = {
      type: undefined,
      logentry: { message: "compare %s with %s", params: [{ a: shared, b: shared }] },
    };

    scrubEventPii(event);

    const param = event.logentry?.params?.[0] as { a: unknown; b: unknown };
    expect(param.a).toEqual({ token: "[REDACTED]", keep: "visible" });
    expect(param.b).toEqual({ token: "[REDACTED]", keep: "visible" });
  });

  it("drops the whole tRPC input when scrubbing it throws", () => {
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

  it("scrubs secrets from exception messages and log entries", () => {
    const event: ErrorEvent = {
      type: undefined,
      message: `captureMessage with ${ROOT_KEY}`,
      logentry: {
        message: "template with %s and %s",
        params: [{ token: ROOT_KEY }, `bare ${JWT}`],
      },
      exception: {
        values: [
          {
            type: "TRPCError",
            value: `Failed to verify key ${ROOT_KEY} against https://api.unkey.com/v1/keys?token=${JWT}`,
          },
        ],
      },
    };

    scrubEventPii(event);

    expect(JSON.stringify(event)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event)).not.toContain(JWT);
    expect(event.exception?.values?.[0]?.type).toBe("TRPCError");
    expect(event.exception?.values?.[0]?.value).toContain("Failed to verify key");
  });

  it("keeps identifiers in messages that carry no secret", () => {
    const event: ErrorEvent = {
      type: undefined,
      exception: {
        values: [{ type: "Error", value: "getDeploymentRuntimeLogs is not a function" }],
      },
    };

    scrubEventPii(event);

    expect(event.exception?.values?.[0]?.value).toBe("getDeploymentRuntimeLogs is not a function");
  });

  it("scrubs extra and request data by key and token shape", () => {
    const event: ErrorEvent = {
      type: undefined,
      extra: {
        password: "hunter2",
        nested: { rootKey: ROOT_KEY, note: `opaque ${JWT}` },
        keep: "plain text",
      },
      request: {
        data: { token: ROOT_KEY, body: `contains ${JWT}` },
      },
    };

    scrubEventPii(event);

    expect(JSON.stringify(event)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event)).not.toContain(JWT);
    expect(event.extra?.password).toBe("[REDACTED]");
    expect(event.extra?.keep).toBe("plain text");
  });

  it("drops extra wholesale when scrubbing it throws", () => {
    const event: ErrorEvent = {
      type: undefined,
      extra: {
        get boom(): string {
          throw new Error("getter exploded");
        },
      },
      request: { url: `/keys?key=${ROOT_KEY}` },
    };

    scrubEventPii(event);

    expect(event.extra).toBeUndefined();
    expect(event.request?.url).not.toContain(ROOT_KEY);
  });
});

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
    expect(event.spans?.[0]?.data["http.method"]).toBe("GET");
    expect(event.spans?.[0]?.data["http.url"]).toContain("https://api.unkey.com/v1/keys");
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
    expect(event.spans?.[0]?.description).toBe("fetch(/api/keys?key=%5BREDACTED%5D)");
    expect(event.spans?.[1]?.description).toBe("redirect url=/login?token=%5BREDACTED%5D");
  });

  it("leaves SQLCommenter blocks in db span descriptions untouched", () => {
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

  it("scrubs the Referer carried as an http.request.header trace attribute", () => {
    const event: TransactionEvent = {
      type: "transaction",
      contexts: {
        trace: {
          span_id: "aaaaaaaaaaaaaaaa",
          trace_id: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
          data: {
            "http.request.header.referer": `https://app.unkey.com/auth/callback?code=${ROOT_KEY}`,
            "http.response.header.location": `https://app.unkey.com/setup?token=${JWT}`,
          },
        },
      },
    };

    scrubTransactionPii(event);

    const data = event.contexts?.trace?.data;
    expect(JSON.stringify(data)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(data)).not.toContain(JWT);
    expect(data?.["http.request.header.referer"]).toContain("/auth/callback");
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
    expect(returned.description).toBe("body > div#root > button.submit");
    expect(returned.data["sentry.exclusive_time"]).toBe(120);
  });

  it("masks SQL literal values in db statement attributes while keeping query shape", () => {
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
    const span = makeSpan({ data: { transaction: `/share/${ROOT_KEY}` } });
    Object.freeze(span.data);
    Object.freeze(span);

    const returned = scrubSpanPii(span);

    expect(JSON.stringify(returned)).not.toContain(ROOT_KEY);
  });
});

describe("scrubLog", () => {
  it("scrubs the raw error message the error filter routes into logs", () => {
    const log: SentryLog = {
      level: "error",
      message: "tRPC operation completed with expected error",
      attributes: {
        trpc_error_message: `Failed to verify key ${ROOT_KEY}`,
        trpc_code: "INTERNAL_SERVER_ERROR",
        user_email: "jane@acme.com",
      },
    };

    const scrubbed = scrubLog(log);

    expect(JSON.stringify(scrubbed)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(scrubbed)).not.toContain("jane@acme.com");
    expect(scrubbed?.attributes?.trpc_code).toBe("INTERNAL_SERVER_ERROR");
  });

  it("drops the log when scrubbing throws", () => {
    const log: SentryLog = {
      level: "error",
      message: "boom",
      attributes: {
        get exploding(): string {
          throw new Error("getter exploded");
        },
      },
    };

    expect(scrubLog(log)).toBeNull();
  });
});

describe("scrubReplayFrame", () => {
  it("scrubs console frames, including the raw arguments alongside the message", () => {
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "breadcrumb",
        payload: {
          type: "default",
          category: "console",
          level: "error",
          message: `Tree layout error: ${ROOT_KEY}`,
          data: { logger: "console", arguments: ["Tree layout error:", { message: ROOT_KEY }] },
        },
      },
    } as unknown as ReplayFrameEvent;

    const scrubbed = scrubReplayFrame(frame);

    expect(JSON.stringify(scrubbed)).not.toContain(ROOT_KEY);
  });

  it("keeps console message and arguments consistent under one policy", () => {
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "breadcrumb",
        payload: {
          type: "default",
          category: "console",
          level: "error",
          message: "request failed INTERNAL_SERVER_ERROR",
          data: { logger: "console", arguments: ["request failed", "INTERNAL_SERVER_ERROR"] },
        },
      },
    } as unknown as ReplayFrameEvent;

    scrubReplayFrame(frame);

    const payload = (frame.data as { payload: { message: string; data: { arguments: string[] } } })
      .payload;
    expect(payload.message).toContain("INTERNAL_SERVER_ERROR");
    expect(payload.data.arguments).toContain("INTERNAL_SERVER_ERROR");
  });

  it("survives a cyclic console argument instead of dropping the frame", () => {
    const cyclic: Record<string, unknown> = { token: ROOT_KEY };
    cyclic.self = cyclic;
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "breadcrumb",
        payload: {
          type: "default",
          category: "console",
          level: "error",
          message: "boom",
          data: { logger: "console", arguments: [cyclic] },
        },
      },
    } as unknown as ReplayFrameEvent;

    const scrubbed = scrubReplayFrame(frame);

    expect(scrubbed).toBe(frame);
    expect(JSON.stringify(scrubbed)).not.toContain(ROOT_KEY);
  });

  it("scrubs the URL a navigation/resource span carries in its description", () => {
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "performanceSpan",
        payload: {
          op: "resource.fetch",
          description: `https://api.unkey.com/v1/keys.verifyKey?token=${JWT}`,
          startTimestamp: 0,
          endTimestamp: 1,
          data: {},
        },
      },
    } as unknown as ReplayFrameEvent;

    scrubReplayFrame(frame);

    expect(JSON.stringify(frame)).not.toContain(JWT);
  });

  it("scrubs URL data fields on navigation breadcrumb frames", () => {
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "breadcrumb",
        payload: {
          type: "default",
          category: "navigation",
          data: { from: `/auth/callback?code=${ROOT_KEY}`, to: "/dashboard" },
        },
      },
    } as unknown as ReplayFrameEvent;

    scrubReplayFrame(frame);

    expect(JSON.stringify(frame)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(frame)).toContain("/dashboard");
  });

  it("leaves ui.click selector paths untouched", () => {
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "breadcrumb",
        payload: {
          type: "default",
          category: "ui.click",
          message: "div > button.Button_root__1a2b3",
        },
      },
    } as unknown as ReplayFrameEvent;

    scrubReplayFrame(frame);

    const message = (frame.data as { payload: { message: string } }).payload.message;
    expect(message).toBe("div > button.Button_root__1a2b3");
  });
});

describe("event-path parity with the transaction and replay paths", () => {
  it("scrubs console breadcrumb data.arguments, not just the formatted message", () => {
    const event: ErrorEvent = {
      type: undefined,
      breadcrumbs: [
        {
          category: "console",
          message: "Failed to log to Sentry: [object Object]",
          data: {
            logger: "console",
            arguments: [
              "Failed to log to Sentry:",
              { attributes: { trpc_error_message: `Failed to verify key ${ROOT_KEY}` } },
            ],
          },
        },
      ],
    };

    scrubEventPii(event);

    expect(JSON.stringify(event.breadcrumbs)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).toContain("trpc_error_message");
  });

  it("scrubs event.transaction, which carries the route just like request.url", () => {
    const event: ErrorEvent = {
      type: undefined,
      transaction: `/keys/${ROOT_KEY}/settings`,
      request: { url: `https://app.unkey.com/keys/${ROOT_KEY}/settings` },
    };

    scrubEventPii(event);

    expect(event.transaction).not.toContain(ROOT_KEY);
    expect(event.request?.url).not.toContain(ROOT_KEY);
    expect(event.transaction).toContain("/settings");
  });

  it("scrubs contexts.trace.data on error events, as the transaction path does", () => {
    const event: ErrorEvent = {
      type: undefined,
      contexts: {
        trace: {
          trace_id: "abc",
          span_id: "def",
          data: {
            "http.url": `https://api.unkey.com/verify?token=${JWT}`,
            "db.query.text": "SELECT * FROM keys WHERE email = 'jane@acme.com'",
          },
        },
      },
    };

    scrubEventPii(event);

    const traceData = event.contexts?.trace?.data;
    expect(JSON.stringify(traceData)).not.toContain(JWT);
    expect(JSON.stringify(traceData)).not.toContain("jane@acme.com");
    expect(traceData?.["db.query.text"]).toContain("SELECT * FROM keys WHERE email =");
  });
});

describe("redaction-net coverage gaps", () => {
  it("redacts emails in URL path segments", () => {
    const out = scrubUrl("/api/users/jane.doe@acme.com/keys");

    expect(out).not.toContain("jane.doe@acme.com");
    expect(out).toContain("/api/users/");
    expect(out).toContain("/keys");
  });

  it("scrubs bracketed (nested) form-encoded bodies by key", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: { data: "user[password]=hunter2&user[email]=jane@acme.com&user[name]=Jane" },
    };

    scrubEventPii(event);

    const data = String(event.request?.data);
    expect(data).not.toContain("hunter2");
    expect(data).not.toContain("jane@acme.com");
    expect(data).toContain("Jane");
  });

  it("redacts credential headers outside the exact allowlist", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: {
        headers: {
          "x-unkey-key": ROOT_KEY,
          "x-csrf-token": "short",
          "x-clerk-session": "sess_abc",
          "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
          accept: "application/json",
        },
      },
    };

    scrubEventPii(event);

    const headers = event.request?.headers ?? {};
    expect(headers["x-unkey-key"]).toBe("[REDACTED]");
    expect(headers["x-csrf-token"]).toBe("[REDACTED]");
    expect(headers["x-clerk-session"]).toBe("[REDACTED]");
    // Benign headers survive intact.
    expect(headers["user-agent"]).toBe("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)");
    expect(headers.accept).toBe("application/json");
  });

  it("scrubs Errors and Maps reaching un-normalized log surfaces", () => {
    const log: SentryLog = {
      level: "error",
      message: "operation failed",
      attributes: {
        cause: new Error(`Failed to verify key ${ROOT_KEY}`),
        lookup: new Map([["contact", "jane@acme.com"]]),
      },
    };

    const scrubbed = scrubLog(log);

    expect(JSON.stringify(scrubbed)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(scrubbed)).not.toContain("jane@acme.com");
    expect(JSON.stringify(scrubbed)).toContain("Error");
  });
});

describe("scrubLog correlation identifiers", () => {
  it("preserves the ids structured logging exists to provide", () => {
    const log: SentryLog = {
      level: "error",
      message: "tRPC operation completed with expected error",
      attributes: {
        service: "dashboard",
        request_id: "req_1763648000000_k3j9xq2mz",
        user_id: "user_2nQ9xPz1AbCdEfGhIjKlMnOpQr",
        workspace_id: "ws_3ZjKcT9pQxV2mNbW",
        version: "a1b2c3d4e5f67890a1b2c3d4e5f67890",
        trpc_procedure: "key.create",
        trpc_error_code: "INTERNAL_SERVER_ERROR",
        trpc_error_message: `Failed to verify key ${ROOT_KEY}`,
      },
    };

    const scrubbed = scrubLog(log);
    const attributes = scrubbed?.attributes ?? {};

    expect(attributes.request_id).toBe("req_1763648000000_k3j9xq2mz");
    expect(attributes.user_id).toBe("user_2nQ9xPz1AbCdEfGhIjKlMnOpQr");
    expect(attributes.workspace_id).toBe("ws_3ZjKcT9pQxV2mNbW");
    expect(attributes.version).toBe("a1b2c3d4e5f67890a1b2c3d4e5f67890");
    expect(attributes.trpc_procedure).toBe("key.create");
    expect(attributes.trpc_error_code).toBe("INTERNAL_SERVER_ERROR");
    expect(String(attributes.trpc_error_message)).not.toContain(ROOT_KEY);
  });

  it("scrubs a templated (fmt) message, which is a String object at runtime", () => {
    const templated = new String(
      `fetch failed for /api/keys?keyId=${ROOT_KEY}`,
    ) as unknown as SentryLog["message"];

    const scrubbed = scrubLog({
      level: "error",
      message: templated,
      attributes: {
        "sentry.message.template": "fetch failed for /api/keys?keyId=%s",
        "sentry.message.parameter.0": ROOT_KEY,
      },
    });

    expect(String(scrubbed?.message)).not.toContain(ROOT_KEY);
    expect(String(scrubbed?.attributes?.["sentry.message.parameter.0"])).not.toContain(ROOT_KEY);
    expect(scrubbed?.attributes?.["sentry.message.template"]).toBe(
      "fetch failed for /api/keys?keyId=%s",
    );
  });
});

describe("over-redaction and encoded-form regressions", () => {
  it("scrubs caller-supplied PII arriving under correlation attribute names", () => {
    const log: SentryLog = {
      level: "info",
      message: "user action: invite.send",
      attributes: {
        resource_id: "jane@acme.com",
        action_type: "invite jane@acme.com",
        request_id: "req_1763648000000_k3j9xq2mz",
      },
    };

    const scrubbed = scrubLog(log);

    expect(JSON.stringify(scrubbed)).not.toContain("jane@acme.com");
    expect(scrubbed?.attributes?.request_id).toBe("req_1763648000000_k3j9xq2mz");
  });

  it("redacts Map values by key instead of flattening keys away", () => {
    const log: SentryLog = {
      level: "error",
      message: "x",
      attributes: { ctx: new Map<string, string>([["password", "hunter2"]]) },
    };

    const scrubbed = scrubLog(log);

    expect(JSON.stringify(scrubbed)).not.toContain("hunter2");
    expect(JSON.stringify(scrubbed)).toContain("password");
  });

  it("redacts percent-encoded emails in URL path segments", () => {
    const out = scrubUrl("https://app.unkey.com/api/users/jane%40acme.com/keys");

    expect(out).not.toContain("jane%40acme.com");
    expect(out).toContain("/api/users/");
    expect(out).toContain("/keys");
  });

  it("preserves correlation and infrastructure headers", () => {
    const event: ErrorEvent = {
      type: undefined,
      request: {
        headers: {
          ":authority": "app.unkey.com",
          "x-request-id": "123e4567-e89b-12d3-a456-426614174000",
          traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
          "sentry-trace": "4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1",
          "x-unkey-key": ROOT_KEY,
        },
      },
    };

    scrubEventPii(event);

    const headers = event.request?.headers ?? {};
    expect(headers[":authority"]).toBe("app.unkey.com");
    expect(headers["x-request-id"]).toBe("123e4567-e89b-12d3-a456-426614174000");
    expect(headers.traceparent).toBe("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01");
    expect(headers["sentry-trace"]).toBe("4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1");
    expect(headers["x-unkey-key"]).toBe("[REDACTED]");
  });

  it("leaves bracketed UI-state params alone while catching credential leaves", () => {
    const out = scrubUrl(
      "https://app.unkey.com/logs?filters[state]=open&sort[key]=name&user[password]=hunter2&user[email]=j@a.co",
    );

    expect(out).toContain("open");
    expect(out).toContain("name");
    expect(out).not.toContain("hunter2");
    expect(out).not.toContain("j%40a.co");
  });

  it("redacts array params named for credentials (token[])", () => {
    const out = scrubUrl("https://app.unkey.com/verify?token[]=abc123&token[]=def456");

    expect(out).not.toContain("abc123");
    expect(out).not.toContain("def456");
  });

  it("keeps Error stack frames readable and carries the non-enumerable cause", () => {
    const inner = new Error(`vault sealed for key ${ROOT_KEY}`);
    const outer = new Error("teardown failed", { cause: inner });
    const log: SentryLog = { level: "error", message: "x", attributes: { e: outer } };

    const scrubbed = scrubLog(log);
    const flat = scrubbed?.attributes?.e as {
      stack?: string;
      cause?: { message?: string };
    };

    expect(flat.cause?.message).toBeDefined();
    expect(JSON.stringify(scrubbed)).not.toContain(ROOT_KEY);
    const frames = (flat.stack ?? "").split("\n").filter((line) => /^\s+at\s/.test(line));
    expect(frames.length).toBeGreaterThan(0);
    for (const frame of frames) {
      expect(frame).not.toContain("[REDACTED]");
    }
  });

  it("gives replay ui.* selector messages the same email pass as the event path", () => {
    const frame = {
      type: 5,
      timestamp: 0,
      data: {
        tag: "breadcrumb",
        payload: {
          type: "default",
          category: "ui.click",
          message: 'div > button[id="jane@acme.com"]',
          timestamp: 0,
        },
      },
    } as unknown as ReplayFrameEvent;

    const scrubbed = scrubReplayFrame(frame);

    const message = (scrubbed?.data as { payload: { message: string } }).payload.message;
    expect(message).not.toContain("jane@acme.com");
    expect(message).toContain("div > button");
  });
});
