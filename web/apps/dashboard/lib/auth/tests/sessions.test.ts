import { beforeEach, describe, expect, it, vi } from "vitest";
import { mockWorkOSEnv } from "../__mocks__/env";

// Mock env before importing the module under test so the WorkOS (non-local)
// branch of updateSession runs.
vi.mock("@/lib/env", () => ({
  env: vi.fn(() => mockWorkOSEnv()),
}));

// The auth provider is a server-initialized singleton. Replace it with mocks so
// we can drive validateSession/refreshSession outcomes directly.
const validateSession = vi.fn();
const refreshSession = vi.fn();
vi.mock("../server", () => ({
  auth: {
    validateSession: (token: string) => validateSession(token),
    refreshSession: (token: string) => refreshSession(token),
  },
}));

const getCookie = vi.fn();
const setSessionCookie = vi.fn();
vi.mock("../cookies", () => ({
  getCookie: (...args: unknown[]) => getCookie(...args),
  setSessionCookie: (...args: unknown[]) => setSessionCookie(...args),
  getCookieOptionsAsString: vi.fn().mockResolvedValue("Path=/; HttpOnly; SameSite=Lax"),
}));

vi.mock("../cookie-security", () => ({
  getAuthCookieOptions: vi.fn().mockReturnValue({}),
}));

import { updateSession } from "../sessions";
import { UNKEY_SESSION_COOKIE } from "../types";

const SESSION_HEADER = `x-${UNKEY_SESSION_COOKIE}`;

describe("updateSession", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getCookie.mockResolvedValue("existing-session-token");
    setSessionCookie.mockResolvedValue(undefined);
  });

  it("returns the refreshed session instead of dropping it", async () => {
    // Guarantees the regression in ENG-2929 stays fixed: when a session is
    // refreshed successfully, updateSession must surface that session rather
    // than falling through to a null session and forcing a sign-out.
    validateSession.mockResolvedValue({ isValid: false, shouldRefresh: true });
    refreshSession.mockResolvedValue({
      newToken: "refreshed-token",
      expiresAt: new Date("2099-01-01T00:00:00Z"),
      session: {
        userId: "user_123",
        orgId: "org_123",
        accessToken: "access_123",
        role: "admin",
        user: null,
      },
    });

    const result = await updateSession();

    expect(result.session).not.toBeNull();
    expect(result.session?.userId).toBe("user_123");
    expect(result.session?.orgId).toBe("org_123");
    expect(result.headers.get(SESSION_HEADER)).toBe("refreshed-token");
    expect(setSessionCookie).toHaveBeenCalledWith({
      token: "refreshed-token",
      expiresAt: new Date("2099-01-01T00:00:00Z"),
    });
  });

  it("does not write a fresh session cookie when refresh yields no session", async () => {
    // The spurious-logout fingerprint: the old code wrote a brand new session
    // cookie and then reported the user as logged out. A refresh that produces
    // no session must be treated as a failure, leaving no cookie behind so the
    // user's login state and cookie state stay consistent.
    validateSession.mockResolvedValue({ isValid: false, shouldRefresh: true });
    refreshSession.mockResolvedValue({
      newToken: "refreshed-token",
      expiresAt: new Date("2099-01-01T00:00:00Z"),
      session: null,
    });

    const result = await updateSession();

    expect(result.session).toBeNull();
    expect(result.headers.get("Set-Cookie")).toBeNull();
    expect(result.headers.get(SESSION_HEADER)).toBeNull();
    expect(setSessionCookie).not.toHaveBeenCalled();
  });
});
