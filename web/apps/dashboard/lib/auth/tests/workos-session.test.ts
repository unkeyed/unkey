import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  withAuth: vi.fn(),
  logOperation: vi.fn(),
  requestHeaders: new Headers(),
  requestCookies: new Set<string>(),
}));

vi.mock("react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react")>()),
  // cache() memoizes per request; each test needs its own resolution.
  cache: (fn: unknown) => fn,
}));
vi.mock("@/lib/env", () => ({ workosAuthEnv: () => ({ WORKOS_COOKIE_NAME: "wos-session" }) }));
vi.mock("@/lib/logging", () => ({ logOperation: mocks.logOperation }));
vi.mock("@workos-inc/authkit-nextjs", () => ({ withAuth: mocks.withAuth }));
vi.mock("next/headers", () => ({
  headers: async () => mocks.requestHeaders,
  cookies: async () => ({ has: (name: string) => mocks.requestCookies.has(name) }),
}));

import { getWorkOSSession } from "../workos-session";

const signedIn = { user: { id: "user_123" }, sessionId: "sess_1", accessToken: "token" };

describe("getWorkOSSession", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.requestHeaders = new Headers();
    mocks.requestCookies = new Set();
    mocks.withAuth.mockResolvedValue(signedIn);
  });

  it("reads the session when the middleware covered the request", async () => {
    mocks.requestHeaders.set("x-workos-middleware", "1");

    await expect(getWorkOSSession()).resolves.toEqual(signedIn);
    expect(mocks.withAuth).toHaveBeenCalledOnce();
  });

  it("renders a missing static file as signed out instead of throwing", async () => {
    // No middleware header: the matcher skips extension paths, so Next renders
    // the 404 through the root layout with no AuthKit context.
    await expect(getWorkOSSession()).resolves.toEqual({ user: null });
    expect(mocks.withAuth).not.toHaveBeenCalled();
  });

  it("stays quiet when the uncovered caller had no session to lose", async () => {
    await getWorkOSSession();

    expect(mocks.logOperation).toHaveBeenCalledWith(
      "debug",
      expect.any(String),
      expect.objectContaining({ auth_carried_session: false }),
    );
  });

  it("warns when a session-bearing request was downgraded to anonymous", async () => {
    mocks.requestCookies.add("wos-session");

    await getWorkOSSession();

    expect(mocks.logOperation).toHaveBeenCalledWith(
      "warn",
      expect.any(String),
      expect.objectContaining({ auth_carried_session: true }),
    );
  });
});
