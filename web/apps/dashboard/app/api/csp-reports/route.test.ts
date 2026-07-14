import { type Mock, afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const getClientMock = vi.fn();

vi.mock("@sentry/nextjs", () => ({
  getClient: () => getClientMock(),
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

function reportRequest(body: string): Request {
  return new Request("http://dashboard.test/api/csp-reports", {
    method: "POST",
    headers: { "Content-Type": "application/csp-report" },
    body,
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
    // Infinity, which must not pass the number allowlist.
    const body = `{"csp-report":{"document-uri":"https://app.unkey.com/","status-code":1e999,"violated-directive":"${"x".repeat(10_000)}"}}`;
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

  it("stops forwarding after the per-window rate limit", async () => {
    const body = JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } });
    for (let i = 0; i < 60; i++) {
      const response = await post(reportRequest(body));
      expect(response.status).toBe(204);
    }
    expect(fetchMock).toHaveBeenCalledTimes(60);

    const throttled = await post(reportRequest(body));
    expect(throttled.status).toBe(204);
    expect(fetchMock).toHaveBeenCalledTimes(60);
  });

  it("enforces the limit over a sliding window, not a resettable fixed window", async () => {
    // Fake only Date so async promise plumbing keeps running on real timers.
    vi.useFakeTimers({ toFake: ["Date"] });
    try {
      const body = JSON.stringify({ "csp-report": { "document-uri": "http://x.test/" } });

      vi.setSystemTime(0);
      for (let i = 0; i < 30; i++) {
        await post(reportRequest(body));
      }
      vi.setSystemTime(50_000);
      for (let i = 0; i < 30; i++) {
        await post(reportRequest(body));
      }
      expect(fetchMock).toHaveBeenCalledTimes(60);

      // At t=62s the 30 forwards from t=0 have aged out but the 30 from
      // t=50s still count: only 30 more may forward. A fixed window that
      // resets to zero at t=60s would allow 60 here — 90 forwards within
      // the span [50s, 62s].
      vi.setSystemTime(62_000);
      for (let i = 0; i < 31; i++) {
        await post(reportRequest(body));
      }
      expect(fetchMock).toHaveBeenCalledTimes(90);
    } finally {
      vi.useRealTimers();
    }
  });
});
