import * as Sentry from "@sentry/nextjs";
import { NextRequest, NextResponse } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getCookie, setCookiesOnResponse } from "../cookies";
import { UNKEY_SESSION_COOKIE } from "../types";

vi.mock("@sentry/nextjs", () => ({
  captureMessage: vi.fn(),
}));

const options = {
  httpOnly: true,
  secure: true,
  sameSite: "lax" as const,
  path: "/",
};

function requestWithResponseCookies(response: NextResponse): NextRequest {
  const cookie = response.headers
    .getSetCookie()
    .filter((header) => !header.includes("Max-Age=0"))
    .map((header) => header.slice(0, header.indexOf(";")))
    .join("; ");

  return new NextRequest("https://app.unkey.com/apis", {
    headers: { cookie },
  });
}

describe("session cookies", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("stores and reads a session larger than one browser cookie", async () => {
    const session = "s".repeat(9000);
    const response = NextResponse.next();

    await setCookiesOnResponse(response, [{ name: UNKEY_SESSION_COOKIE, value: session, options }]);

    const headers = response.headers.getSetCookie();
    expect(headers).toHaveLength(5);
    expect(headers).toEqual([
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}=;`),
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}.0=`),
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}.1=`),
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}.2=`),
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}.3=;`),
    ]);
    expect(headers.every((header) => header.length < 4096)).toBe(true);
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      "WorkOS session cookie exceeds browser size limit",
      {
        level: "warning",
        fingerprint: ["workos-session-cookie-oversized"],
        tags: { component: "authentication" },
        extra: { sessionSizeBytes: 9000, chunkCount: 3 },
      },
    );

    const request = requestWithResponseCookies(response);
    await expect(getCookie(UNKEY_SESSION_COOKIE, request)).resolves.toBe(session);
  });

  it("keeps small sessions compatible with the original cookie name", async () => {
    const response = NextResponse.next();

    await setCookiesOnResponse(response, [
      { name: UNKEY_SESSION_COOKIE, value: "small-session", options },
    ]);

    expect(response.headers.getSetCookie()).toEqual([
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}=small-session;`),
      expect.stringContaining(`${UNKEY_SESSION_COOKIE}.0=;`),
    ]);
    expect(Sentry.captureMessage).not.toHaveBeenCalled();

    const request = requestWithResponseCookies(response);
    await expect(getCookie(UNKEY_SESSION_COOKIE, request)).resolves.toBe("small-session");
  });

  it("rejects sessions that exceed the bounded cookie set", async () => {
    const response = NextResponse.next();

    await expect(
      setCookiesOnResponse(response, [
        { name: UNKEY_SESSION_COOKIE, value: "s".repeat(28_001), options },
      ]),
    ).rejects.toThrow("Session is too large to store in cookies");
  });
});
