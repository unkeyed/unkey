import { type Mock, afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const getClientMock = vi.fn();
const captureMessageMock = vi.fn();

vi.mock("@sentry/nextjs", () => ({
  getClient: () => getClientMock(),
  captureMessage: (message: string, level: string) => captureMessageMock(message, level),
}));

// The route defers the Sentry forward with after(); run the task immediately
// so tests can assert on the fetch mock right after POST resolves. (The
// task's fetch call happens synchronously at invocation; the route swallows
// its rejections itself.)
vi.mock("next/server", () => ({
  after: (task: () => unknown): void => {
    void task();
  },
}));

type FetchMock = Mock<Parameters<typeof fetch>, Promise<Response>>;
type PostHandler = (request: Request) => Promise<Response>;

const SENTRY_DSN = {
  protocol: "https",
  publicKey: "publickey123",
  host: "o123.ingest.us.sentry.io",
  port: "",
  path: "",
  projectId: "456",
};

function reportRequest(body: string, headers?: Record<string, string>): Request {
  return new Request("http://dashboard.test/api/csp-reports", {
    method: "POST",
    headers: headers ?? { "Content-Type": "application/csp-report" },
    body,
  });
}

/**
 * A report whose violation signature (directive + blocked origin) is unique to
 * `index`, so a sequence of them exercises the global forward budget rather
 * than the per-signature one.
 */
function distinctViolation(index: number): string {
  return JSON.stringify({
    "csp-report": {
      "document-uri": "https://app.unkey.com/",
      "effective-directive": "script-src",
      "blocked-uri": `https://blocked-${index}.example/x.js`,
    },
  });
}

function forwardedReport(fetchMock: FetchMock): Record<string, unknown> {
  const [, init] = fetchMock.mock.calls[0];
  const body = init?.body;
  if (typeof body !== "string") {
    throw new Error("expected fetch to be called with a string body");
  }
  const parsed: unknown = JSON.parse(body);
  if (
    typeof parsed !== "object" ||
    parsed === null ||
    !("csp-report" in parsed) ||
    typeof parsed["csp-report"] !== "object" ||
    parsed["csp-report"] === null
  ) {
    throw new Error("expected forwarded body to wrap a csp-report object");
  }
  return { ...parsed["csp-report"] };
}

describe("POST /api/csp-reports", () => {
  let post: PostHandler;
  let fetchMock: FetchMock;

  beforeEach(async () => {
    // Fresh module per test so the module-level rate-limiter window never
    // couples tests to each other's forward counts.
    vi.resetModules();
    ({ POST: post } = await import("./route"));

    getClientMock.mockReturnValue({ getDsn: () => SENTRY_DSN });
    fetchMock = vi.fn<Parameters<typeof fetch>, Promise<Response>>();
    fetchMock.mockResolvedValue(new Response(null, { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("drops reports without forwarding when Sentry is not initialized", async () => {
    getClientMock.mockReturnValue(undefined);

    const response = await post(
      reportRequest(JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } })),
    );

    expect(response.status).toBe(204);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("scrubs bearer credentials from URL fields before forwarding", async () => {
    const token = "abcdefghijklmnopqrstuvwx123";
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": `https://app.unkey.com/auth/sign-up?invitation_token=${token}`,
            "blocked-uri": "https://evil.example/payload.js",
            "violated-directive": "script-src",
            "script-sample": "alert(document.cookie)",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const [endpoint] = fetchMock.mock.calls[0];
    expect(endpoint).toBe(
      "https://o123.ingest.us.sentry.io/api/456/security/?sentry_key=publickey123",
    );

    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["document-uri"]).not.toContain(token);
    expect(forwarded["document-uri"]).toContain("/auth/sign-up");
    expect(forwarded["blocked-uri"]).toBe("https://evil.example/payload.js");
    expect(forwarded["violated-directive"]).toBe("script-src");
    expect(forwarded).not.toHaveProperty("script-sample");
  });

  it("drops the query string of non-http(s) URL fields wholesale", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": "https://app.unkey.com/apis",
            "blocked-uri": "wss://app.unkey.com/ws?password=hunter2&email=bob@example.com",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["blocked-uri"]).toBe("wss://app.unkey.com/ws?[REDACTED]");
  });

  it("drops the fragment of non-http(s) URL fields wholesale", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "blocked-uri": "wss://app.unkey.com/ws#password=hunter2",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["blocked-uri"]).toBe("wss://app.unkey.com/ws#[REDACTED]");
  });

  it("redacts short invitation tokens by param name, not just token shape", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": "https://app.unkey.com/join?invitation_token=short7",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["document-uri"]).not.toContain("short7");
    expect(forwarded["document-uri"]).toContain("/join");
  });

  it("rejects reports with no recognized fields without consuming forward budget", async () => {
    const empty = JSON.stringify({ "csp-report": {} });
    const junk = JSON.stringify({ "csp-report": { "x-unknown": 1 } });

    expect((await post(reportRequest(empty))).status).toBe(400);
    expect((await post(reportRequest(junk))).status).toBe(400);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("wholesale-redacts https-prefixed values that fail URL parsing", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            // Space in the host makes new URL() throw despite the https://
            // prefix; the short named secrets must not survive the fallback.
            "document-uri": "https://exa mple.com/path?email=bob@x.com&password=hunter2",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["document-uri"]).toBe("https://exa mple.com/path?[REDACTED]");
  });

  it("scrubs http(s) URL fields that carry leading whitespace", async () => {
    const token = "abcdefghijklmnopqrstuvwx123";
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": ` https://app.unkey.com/join?invitation_token=${token}`,
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["document-uri"]).not.toContain(token);
    expect(forwarded["document-uri"]).toContain("/join");
  });

  it("strips userinfo credentials from non-http(s) URL fields", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "blocked-uri": "wss://alice:hunter2@app.unkey.com/ws",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["blocked-uri"]).not.toContain("alice");
    expect(forwarded["blocked-uri"]).not.toContain("hunter2");
    expect(forwarded["blocked-uri"]).toBe("wss://app.unkey.com/ws");
  });

  it("strips userinfo credentials from https-prefixed values that fail URL parsing", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            // Space in the host makes new URL() throw, forcing the opaque
            // fallback — credentials must not survive it either.
            "document-uri": "https://alice:hunter2@exa mple.com/path",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["document-uri"]).not.toContain("alice");
    expect(forwarded["document-uri"]).not.toContain("hunter2");
    expect(forwarded["document-uri"]).toBe("https://exa mple.com/path");
  });

  it("strips userinfo credentials from http(s) URL fields", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": "https://alice:hunter2@app.unkey.com/apis",
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["document-uri"]).not.toContain("alice");
    expect(forwarded["document-uri"]).not.toContain("hunter2");
    expect(forwarded["document-uri"]).toContain("app.unkey.com/apis");
  });

  it("forwards CSP keyword values unmangled instead of parsing them as URLs", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": "about:blank",
            "blocked-uri": "inline",
            "effective-directive": "script-src-elem",
            "status-code": 200,
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["blocked-uri"]).toBe("inline");
    expect(forwarded["document-uri"]).toBe("about:blank");
    expect(forwarded["status-code"]).toBe(200);
  });

  it("drops unknown fields and wrong-typed values instead of relaying them", async () => {
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "blocked-uri": "https://evil.example/",
            "document-uri": ["https://app.unkey.com/?token=abcdefghijklmnopqrstuvwx123"],
            "line-number": "not-a-number",
            "x-smuggled": { nested: "payload" },
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded["blocked-uri"]).toBe("https://evil.example/");
    expect(forwarded).not.toHaveProperty("document-uri");
    expect(forwarded).not.toHaveProperty("line-number");
    expect(forwarded).not.toHaveProperty("x-smuggled");
  });

  it("drops non-finite numbers and truncates oversized string fields", async () => {
    // Raw JSON so the wire payload carries 1e999 — JSON.parse turns it into
    // Infinity, which must not pass the number allowlist. The padding is
    // "x." repeated so no 20+ char run is token-shaped: this asserts the
    // length cap itself, not the redaction that would otherwise collapse it.
    const padding = "x.".repeat(5_000);
    const body = `{"csp-report":{"document-uri":"https://app.unkey.com/","status-code":1e999,"violated-directive":"${padding}"}}`;
    const response = await post(reportRequest(body));

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    expect(forwarded).not.toHaveProperty("status-code");
    const directive = forwarded["violated-directive"];
    if (typeof directive !== "string") {
      throw new Error("expected violated-directive to be forwarded as a string");
    }
    expect(directive.length).toBe(2_048);
  });

  // Rejecting a content type a real browser sends is invisible — those users'
  // violations silently never reach Sentry — so each browser's media type is
  // pinned as its own case. The accepted types are all non-simple, which is
  // what makes the check load-bearing: a cross-site no-cors flood can only set
  // the simple types below, and this route answers no CORS preflight.
  it.each([
    { contentType: "application/csp-report", sender: "Chrome/Firefox report-uri" },
    { contentType: "application/csp-report; charset=utf-8", sender: "report-uri with a parameter" },
    { contentType: "application/json", sender: "WebKit/Safari report-uri" },
  ])("forwards reports sent as $contentType ($sender)", async ({ contentType }) => {
    const body = JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } });

    const response = await post(reportRequest(body, { "Content-Type": contentType }));

    expect(response.status).toBe(204);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it.each([
    { contentType: "application/x-www-form-urlencoded" },
    { contentType: "text/plain" },
    { contentType: "multipart/form-data" },
  ])("rejects $contentType, which no browser uses for reports", async ({ contentType }) => {
    const body = JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } });

    const response = await post(reportRequest(body, { "Content-Type": contentType }));

    expect(response.status).toBe(415);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  /**
   * The Reporting API (`report-to`) is not configured, and its payload is an
   * array of `{type, body}` envelopes that the parser here cannot read. Pin the
   * rejection so nobody mistakes the media type for working support: enabling
   * report-to means teaching the parser that shape, not just widening the
   * content-type gate.
   */
  it("rejects Reporting API payloads, which it does not know how to parse", async () => {
    const reportToBody = JSON.stringify([
      {
        type: "csp-violation",
        body: { documentURL: "https://app.unkey.com/", effectiveDirective: "script-src" },
      },
    ]);

    const response = await post(
      reportRequest(reportToBody, { "Content-Type": "application/reports+json" }),
    );

    expect(response.status).toBe(415);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("rejects cross-site requests by Fetch Metadata", async () => {
    const response = await post(
      reportRequest(JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } }), {
        "Content-Type": "application/csp-report",
        "Sec-Fetch-Site": "cross-site",
      }),
    );

    expect(response.status).toBe(403);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  // A genuine report POST can carry any of these — "none" when there is no
  // document initiator, "same-site" when the violating page is on a sibling
  // subdomain — and older browsers send no Fetch Metadata at all (covered by
  // every other test here, which omits the header). Dropping any of them would
  // make the rollout silently blind for those users.
  it.each([{ site: "same-origin" }, { site: "same-site" }, { site: "none" }])(
    "forwards reports labelled Sec-Fetch-Site: $site",
    async ({ site }) => {
      const response = await post(
        reportRequest(JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } }), {
          "Content-Type": "application/csp-report",
          "Sec-Fetch-Site": site,
        }),
      );

      expect(response.status).toBe(204);
      expect(fetchMock).toHaveBeenCalledTimes(1);
    },
  );

  it("redacts token-shaped secrets from non-URL string fields", async () => {
    const nonce = "abcdefghijklmnopqrstuvwx123";
    const response = await post(
      reportRequest(
        JSON.stringify({
          "csp-report": {
            "document-uri": "https://app.unkey.com/",
            "original-policy": `script-src 'nonce-${nonce}'; report-uri /api/csp-reports`,
          },
        }),
      ),
    );

    expect(response.status).toBe(204);
    const forwarded = forwardedReport(fetchMock);
    const policy = forwarded["original-policy"];
    if (typeof policy !== "string") {
      throw new Error("expected original-policy to be forwarded as a string");
    }
    expect(policy).not.toContain(nonce);
    expect(policy).toContain("script-src");
  });

  it("rejects bodies that are not csp-report objects", async () => {
    const response = await post(reportRequest(JSON.stringify({ unrelated: true })));

    expect(response.status).toBe(400);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("rejects malformed JSON", async () => {
    const response = await post(reportRequest("not json"));

    expect(response.status).toBe(400);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("rejects oversized bodies without forwarding", async () => {
    const huge = JSON.stringify({
      "csp-report": { "document-uri": `http://x.test/?p=${"a".repeat(20_000)}` },
    });

    const response = await post(reportRequest(huge));

    expect(response.status).toBe(413);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("caps the body in bytes, not UTF-16 code units", async () => {
    // 5,000 astral-plane chars: ~10KB of UTF-16 code units (under the cap if
    // it were measured in string length) but ~20KB of UTF-8 bytes (over the
    // 16KB byte cap) — only byte counting rejects this payload.
    const huge = JSON.stringify({
      "csp-report": { "document-uri": `http://x.test/?p=${"\u{1F600}".repeat(5_000)}` },
    });
    expect(huge.length).toBeLessThan(16_384);
    expect(new TextEncoder().encode(huge).byteLength).toBeGreaterThan(16_384);

    const response = await post(reportRequest(huge));

    expect(response.status).toBe(413);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("still returns 204 when forwarding to Sentry fails", async () => {
    fetchMock.mockRejectedValue(new Error("network down"));

    const response = await post(
      reportRequest(JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } })),
    );

    expect(response.status).toBe(204);
  });

  it("stops forwarding after the global per-window rate limit", async () => {
    for (let i = 0; i < 60; i++) {
      const response = await post(reportRequest(distinctViolation(i)));
      expect(response.status).toBe(204);
    }
    expect(fetchMock).toHaveBeenCalledTimes(60);

    const throttled = await post(reportRequest(distinctViolation(60)));
    expect(throttled.status).toBe(204);
    expect(fetchMock).toHaveBeenCalledTimes(60);
  });

  it("enforces the limit over a sliding window, not a resettable fixed window", async () => {
    // Fake only Date so async promise plumbing keeps running on real timers.
    vi.useFakeTimers({ toFake: ["Date"] });
    try {
      vi.setSystemTime(0);
      for (let i = 0; i < 30; i++) {
        await post(reportRequest(distinctViolation(i)));
      }
      vi.setSystemTime(50_000);
      for (let i = 30; i < 60; i++) {
        await post(reportRequest(distinctViolation(i)));
      }
      expect(fetchMock).toHaveBeenCalledTimes(60);

      // At t=62s the 30 forwards from t=0 have aged out but the 30 from
      // t=50s still count: only 30 more may forward. A fixed window that
      // resets to zero at t=60s would allow 60 here — 90 forwards within
      // the span [50s, 62s].
      vi.setSystemTime(62_000);
      for (let i = 60; i < 91; i++) {
        await post(reportRequest(distinctViolation(i)));
      }
      expect(fetchMock).toHaveBeenCalledTimes(90);
    } finally {
      vi.useRealTimers();
    }
  });

  /**
   * The rollout-critical guarantee: one page repeating a single violation must
   * not spend the global budget and bury the rare violation that would break
   * users once the report-only policy is enforced.
   */
  it("keeps one repeated violation from starving a distinct rare one", async () => {
    const noisy = JSON.stringify({
      "csp-report": {
        "document-uri": "https://app.unkey.com/apis",
        "effective-directive": "script-src",
        "blocked-uri": "https://noisy.example/analytics.js",
      },
    });

    // Far more than the global budget, all one violation.
    for (let i = 0; i < 200; i++) {
      expect((await post(reportRequest(noisy))).status).toBe(204);
    }
    // Capped at the per-signature limit, so the global budget is barely touched.
    expect(fetchMock).toHaveBeenCalledTimes(5);

    const rare = JSON.stringify({
      "csp-report": {
        "document-uri": "https://app.unkey.com/settings",
        "effective-directive": "connect-src",
        "blocked-uri": "https://rare.example/api",
      },
    });
    await post(reportRequest(rare));

    expect(fetchMock).toHaveBeenCalledTimes(6);
    const [, init] = fetchMock.mock.calls[5];
    expect(String(init?.body)).toContain("rare.example");
  });

  it("collapses one violation reported from many paths on the same origin", async () => {
    for (let i = 0; i < 20; i++) {
      const body = JSON.stringify({
        "csp-report": {
          "document-uri": "https://app.unkey.com/apis",
          "effective-directive": "script-src",
          // Same origin, different path: one violation to fix, not twenty.
          "blocked-uri": `https://cdn.example/bundle-${i}.js`,
        },
      });
      await post(reportRequest(body));
    }

    expect(fetchMock).toHaveBeenCalledTimes(5);
  });

  /**
   * A dropped report is answered 204 exactly like a forwarded one, so without
   * this alarm a saturated limiter is indistinguishable from a clean policy —
   * and "the report stream is quiet" is what promotes the policy to enforced.
   */
  it("reports global-budget saturation to Sentry, at most once per window", async () => {
    for (let i = 0; i < 60; i++) {
      await post(reportRequest(distinctViolation(i)));
    }
    expect(captureMessageMock).not.toHaveBeenCalled();

    for (let i = 60; i < 70; i++) {
      await post(reportRequest(distinctViolation(i)));
    }

    expect(captureMessageMock).toHaveBeenCalledTimes(1);
    const [message, level] = captureMessageMock.mock.calls[0];
    expect(message).toContain("saturated");
    expect(level).toBe("warning");
  });

  it("does not alarm on per-signature deduplication, which is not lost signal", async () => {
    const body = JSON.stringify({
      "csp-report": {
        "document-uri": "https://app.unkey.com/apis",
        "effective-directive": "script-src",
        "blocked-uri": "https://noisy.example/a.js",
      },
    });
    for (let i = 0; i < 50; i++) {
      await post(reportRequest(body));
    }

    expect(fetchMock).toHaveBeenCalledTimes(5);
    expect(captureMessageMock).not.toHaveBeenCalled();
  });
});
