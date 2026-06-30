import type { ErrorEvent } from "@sentry/nextjs";
import { describe, expect, it } from "vitest";
import { scrubEventPii, scrubUrl } from "./pii-scrubber";

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
    } as unknown as ErrorEvent;

    scrubEventPii(event);

    expect(event.request?.url).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.request?.query_string)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).not.toContain(ROOT_KEY);
    expect(JSON.stringify(event.breadcrumbs)).not.toContain(JWT);
    // Non-sensitive breadcrumb data is preserved.
    expect(JSON.stringify(event.breadcrumbs)).toContain("/dashboard");
  });

  it("is a no-op on events without request or breadcrumbs", () => {
    const event = { type: undefined } as unknown as ErrorEvent;
    expect(() => scrubEventPii(event)).not.toThrow();
  });
});
