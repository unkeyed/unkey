import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  withAuth: vi.fn(),
  workosAuthEnv: vi.fn(),
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return { ...actual, cache: <T>(fn: T) => fn };
});

vi.mock("@/lib/env", () => ({
  workosAuthEnv: mocks.workosAuthEnv,
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  withAuth: mocks.withAuth,
}));

import { getWorkOSSession } from "./workos-session";

describe("getWorkOSSession", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns the session resolved by AuthKit", async () => {
    const session = { user: { id: "user_123" } };
    mocks.withAuth.mockResolvedValue(session);

    await expect(getWorkOSSession()).resolves.toBe(session);
  });

  it("treats a route outside the AuthKit middleware as unauthenticated", async () => {
    mocks.withAuth.mockRejectedValue(
      new Error(
        "You are calling 'withAuth' on /apple-touch-icon-precomposed.png that isn't covered by the AuthKit middleware. Make sure it is running on all paths you are calling 'withAuth' from by updating your middleware config in 'middleware.(js|ts)'.",
      ),
    );

    await expect(getWorkOSSession()).resolves.toEqual({ user: null });
  });

  it("rethrows unrelated errors", async () => {
    mocks.withAuth.mockRejectedValue(new Error("session decryption failed"));

    await expect(getWorkOSSession()).rejects.toThrow("session decryption failed");
  });
});
