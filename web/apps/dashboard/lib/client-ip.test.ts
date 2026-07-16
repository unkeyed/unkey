import { describe, expect, it } from "vitest";
import { getClientIp } from "./client-ip";

describe("getClientIp", () => {
  it("returns undefined when no forwarding header is present", () => {
    expect(getClientIp(new Headers())).toBeUndefined();
  });

  it("reads the client IP from x-forwarded-for", () => {
    expect(getClientIp(new Headers({ "x-forwarded-for": "203.0.113.7" }))).toBe("203.0.113.7");
  });

  it("takes only the leftmost entry of a forwarding chain", () => {
    expect(
      getClientIp(new Headers({ "x-forwarded-for": "203.0.113.7, 198.51.100.1, 10.0.0.1" })),
    ).toBe("203.0.113.7");
  });

  it("prefers x-vercel-forwarded-for over every other forwarding header", () => {
    const headers = new Headers({
      "x-vercel-forwarded-for": "203.0.113.7",
      "x-forwarded-for": "198.51.100.1",
      "x-real-ip": "192.0.2.1",
    });

    expect(getClientIp(headers)).toBe("203.0.113.7");
  });

  it("prefers x-forwarded-for over x-real-ip", () => {
    expect(
      getClientIp(new Headers({ "x-forwarded-for": "198.51.100.1", "x-real-ip": "192.0.2.1" })),
    ).toBe("198.51.100.1");
  });

  it("falls back to x-real-ip when no other header resolves", () => {
    expect(
      getClientIp(new Headers({ "x-forwarded-for": "not-an-ip", "x-real-ip": "192.0.2.1" })),
    ).toBe("192.0.2.1");
  });

  it("handles IPv6 addresses", () => {
    expect(getClientIp(new Headers({ "x-forwarded-for": "2001:db8::1" }))).toBe("2001:db8::1");
  });

  it("handles the IPv4-mapped IPv6 form a dual-stack socket produces", () => {
    expect(getClientIp(new Headers({ "x-forwarded-for": "::ffff:203.0.113.7" }))).toBe(
      "::ffff:203.0.113.7",
    );
  });

  it.each([
    { name: "IPv4 with a port", value: "203.0.113.7:8080", want: "203.0.113.7" },
    { name: "bracketed IPv6 with a port", value: "[2001:db8::1]:8080", want: "2001:db8::1" },
    { name: "bracketed IPv6 without a port", value: "[2001:db8::1]", want: "2001:db8::1" },
  ])("normalizes $name to a bare address", ({ value, want }) => {
    expect(getClientIp(new Headers({ "x-forwarded-for": value }))).toBe(want);
  });

  // Every one of these would previously have been written to the audit log verbatim as the client's
  // remote_ip. Discarding them is the guarantee ENG-3019 asks for.
  it.each([
    { name: "arbitrary text", value: "not-an-ip" },
    { name: "the literal 'unknown'", value: "unknown" },
    { name: "a SQL payload", value: "'; DROP TABLE audit_log; --" },
    { name: "a log-forging payload", value: 'admin" location="1.2.3.4' },
    { name: "an out-of-range IPv4", value: "999.999.999.999" },
    { name: "an empty value", value: "" },
  ])("discards $name", ({ value }) => {
    expect(getClientIp(new Headers({ "x-forwarded-for": value }))).toBeUndefined();
  });

  it("discards a value carrying a log-injection payload", () => {
    // `Headers` itself rejects control characters, so this goes through a bare header lookup to
    // cover deployments where the header reaches us from something more permissive.
    const headers = {
      get: (name: string) => {
        return name === "x-forwarded-for" ? "203.0.113.7\nlocation=192.0.2.1" : null;
      },
    };

    expect(getClientIp(headers)).toBeUndefined();
  });

  it("does not fall through to a later chain entry when the leftmost one is forged", () => {
    // A client that prepends junk must not be able to have the next entry read as its address.
    expect(getClientIp(new Headers({ "x-forwarded-for": "spoofed, 203.0.113.7" }))).toBeUndefined();
  });

  it("ignores a forged x-forwarded-for when x-vercel-forwarded-for is present", () => {
    const headers = new Headers({
      "x-vercel-forwarded-for": "198.51.100.1",
      "x-forwarded-for": "attacker-supplied",
    });

    expect(getClientIp(headers)).toBe("198.51.100.1");
  });
});
