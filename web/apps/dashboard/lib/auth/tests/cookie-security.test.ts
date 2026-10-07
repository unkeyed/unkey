import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  vercelEnv: "development" as "development" | "preview" | "production",
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ VERCEL_ENV: mocks.vercelEnv }),
}));

import { getAuthCookieOptions, shouldUseSecureCookies } from "../cookie-security";

describe("local session cookie security", () => {
  beforeEach(() => {
    mocks.vercelEnv = "development";
  });

  it.each(["preview", "production"] as const)(
    "uses secure cookies in the %s environment",
    (vercelEnv) => {
      mocks.vercelEnv = vercelEnv;
      expect(shouldUseSecureCookies()).toBe(true);
    },
  );

  it("allows local HTTP development", () => {
    expect(shouldUseSecureCookies()).toBe(false);
    expect(getAuthCookieOptions()).toEqual({
      httpOnly: true,
      secure: false,
      sameSite: "lax",
      path: "/",
    });
  });
});
