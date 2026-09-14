import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authProvider: "workos" as "workos" | "local",
  requestUrl: "https://dashboard-git-auth-no-custom-unkey.vercel.app/settings",
  signOutWithAuthKit: vi.fn(),
}));

vi.mock("next/headers", () => ({
  headers: async () => new Headers({ "x-url": mocks.requestUrl }),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider }),
  workosAuthEnv: vi.fn(),
}));

vi.mock("@/lib/utils", () => ({
  getBaseUrl: () => "https://dashboard.unkey.com",
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  signOut: mocks.signOutWithAuthKit,
}));

vi.mock("next/navigation", () => ({ redirect: vi.fn() }));

import { signOut } from "../utils";

describe("signOut", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authProvider = "workos";
    mocks.requestUrl = "https://dashboard-git-auth-no-custom-unkey.vercel.app/settings";
  });

  it("returns WorkOS logout to the current preview deployment", async () => {
    await signOut();

    expect(mocks.signOutWithAuthKit).toHaveBeenCalledWith({
      returnTo: "https://dashboard-git-auth-no-custom-unkey.vercel.app/auth/sign-in",
    });
  });
});
