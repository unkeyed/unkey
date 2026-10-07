import { describe, expect, it } from "vitest";
import { customDomainSchema, parseDomain } from "./schema";

function domainErrors(domain: string): string[] {
  const result = customDomainSchema.safeParse({ environmentId: "env_1", domain });
  return result.success ? [] : result.error.issues.map((issue) => issue.message);
}

const FORMAT_ERROR = "Enter a domain like api.example.com";

// Cases from TestParseDomain* in pkg/domain/domaingate/domaingate_test.go. The
// client must agree with the server on every one it can decide.
describe("parseDomain", () => {
  it("returns the canonical form the server stores", () => {
    expect(
      Object.fromEntries(
        [
          "api.acme.com",
          "API.ACME.COM",
          "acme.com",
          "a-b.c-d.example.com",
          "münchen.de",
          "MÜNCHEN.DE",
          "xn--mnchen-3ya.de",
          "日本.jp",
          "acme.co.uk",
          "api.acme.co.uk",
          "acme.github.io",
          "acme.notarealtld",
        ].map((input) => [input, parseDomain(input)]),
      ),
    ).toEqual({
      "api.acme.com": { ok: true, domain: "api.acme.com" },
      "API.ACME.COM": { ok: true, domain: "api.acme.com" },
      "acme.com": { ok: true, domain: "acme.com" },
      "a-b.c-d.example.com": { ok: true, domain: "a-b.c-d.example.com" },
      "münchen.de": { ok: true, domain: "xn--mnchen-3ya.de" },
      "MÜNCHEN.DE": { ok: true, domain: "xn--mnchen-3ya.de" },
      "xn--mnchen-3ya.de": { ok: true, domain: "xn--mnchen-3ya.de" },
      "日本.jp": { ok: true, domain: "xn--wgv71a.jp" },
      "acme.co.uk": { ok: true, domain: "acme.co.uk" },
      "api.acme.co.uk": { ok: true, domain: "api.acme.co.uk" },
      "acme.github.io": { ok: true, domain: "acme.github.io" },
      "acme.notarealtld": { ok: true, domain: "acme.notarealtld" },
    });
  });

  it("accepts the RFC 1035 caps of 63 per label and 253 in total", () => {
    const label = "k".repeat(49);
    const longest = `${[label, label, label, label, label].join(".")}.com`;
    expect(longest).toHaveLength(253);
    expect(parseDomain(longest)).toEqual({ ok: true, domain: longest });
    expect(parseDomain(`${"k".repeat(63)}.acme.com`).ok).toBe(true);
  });

  it("rejects what the server rejects as malformed", () => {
    for (const input of [
      "localhost",
      ".acme.com",
      "acme.com.",
      "a..com",
      "-api.acme.com",
      "api-.acme.com",
      "api_v2.acme.com",
      "api.acme.c",
      "https://api.acme.com",
      "api.acme.com/v1",
      "api.acme.com:8080",
      "api acme.com",
      "*.acme.com",
      "xn--a.com",
      `${"a".repeat(250)}.com`,
      `${"kebap".repeat(13)}.acme.com`,
      `acme.${"k".repeat(64)}`,
      "acme.com。",
      `k${[1, 2, 3, 4, 5].map(() => "k".repeat(49)).join(".")}.com`,
    ]) {
      expect(parseDomain(input), input).toMatchObject({ ok: false });
    }
  });

  it("rejects IP-like names", () => {
    for (const input of [
      "1.2.3.4",
      "127.0.0.1",
      "012.0.0.1",
      "0x7f.0.0.1",
      "acme.0x7f",
      "acme.1",
    ]) {
      expect(parseDomain(input), input).toMatchObject({ ok: false });
    }
  });

  it("rejects a reserved double hyphen outside a Punycode label", () => {
    expect(parseDomain("ab--cd.acme.com")).toEqual({ ok: false, error: FORMAT_ERROR });
    expect(parseDomain("exa--mple.com").ok).toBe(true);
  });

  it("accepts final labels the server accepts", () => {
    expect(parseDomain("acme.c0m").ok).toBe(true);
    expect(parseDomain("acme.0xg").ok).toBe(true);
    expect(parseDomain("shop.xn--p1ai").ok).toBe(true);
  });
});

describe("customDomainSchema", () => {
  it("trims and canonicalizes the domain", () => {
    expect(
      customDomainSchema.parse({ environmentId: "env_1", domain: " API.München.de " }).domain,
    ).toBe("api.xn--mnchen-3ya.de");
  });

  it("names the length limit that failed", () => {
    expect(domainErrors(`${"a".repeat(64)}.example.com`)).toEqual([
      "Each part between dots must be at most 63 characters",
    ]);
    const domain = `${"a".repeat(63)}.${"b".repeat(63)}.${"c".repeat(63)}.${"d".repeat(58)}.com`;
    expect(domainErrors(domain)).toEqual(["Domain must be at most 253 characters"]);
  });

  it("keeps the format error for malformed input", () => {
    expect(domainErrors("not a domain")).toEqual([FORMAT_ERROR]);
    expect(domainErrors("example.c")).toEqual([FORMAT_ERROR]);
    expect(domainErrors("example.xn--")).toEqual([FORMAT_ERROR]);
    expect(domainErrors("localhost")).toEqual([FORMAT_ERROR]);
  });

  it("requires a domain", () => {
    expect(domainErrors("  ")).toEqual(["Domain is required"]);
  });
});
