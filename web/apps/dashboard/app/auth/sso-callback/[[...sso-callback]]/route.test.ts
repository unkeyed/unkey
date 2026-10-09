import { NextRequest, NextResponse } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authProvider: "workos" as "workos" | "local",
  baseURL: undefined as string | undefined,
  handleAuth: vi.fn(),
  logManagedAuthOutcome: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider, DASHBOARD_BASE_URL: mocks.baseURL }),
  workosAuthEnv: vi.fn(),
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  handleAuth: mocks.handleAuth,
}));

vi.mock("@/lib/auth/telemetry", () => ({
  logManagedAuthOutcome: mocks.logManagedAuthOutcome,
}));

import { GET } from "./route";

const origins = [
  { baseURL: undefined, expectedOrigin: "http://localhost:3000" },
  {
    baseURL: "https://dashboard.example.test:3443",
    expectedOrigin: "https://dashboard.example.test:3443",
  },
];

describe("AuthKit callback", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authProvider = "workos";
    mocks.baseURL = undefined;
    mocks.handleAuth.mockImplementation(
      (options: {
        baseURL?: string;
        onSuccess: () => void | Promise<void>;
        onError: (params: { request: NextRequest }) => Response | Promise<Response>;
      }) =>
        async (request: NextRequest) => {
          if (request.nextUrl.searchParams.has("error")) {
            return options.onError({ request });
          }
          await options.onSuccess();
          return NextResponse.redirect(new URL("/apis", options.baseURL ?? request.url));
        },
    );
  });

  it.each(origins)(
    "returns AuthKit success to $expectedOrigin and clears the legacy cookie",
    async ({ baseURL, expectedOrigin }) => {
      mocks.baseURL = baseURL;
      const response = await GET(
        new NextRequest("http://localhost:3000/auth/sso-callback?code=code&state=state", {
          headers: { cookie: "unkey-session=legacy" },
        }),
      );

      expect(mocks.handleAuth).toHaveBeenCalledOnce();
      expect(response.headers.get("location")).toBe(`${expectedOrigin}/apis`);
      expect(response.headers.get("set-cookie")).toContain("unkey-session=");
      expect(response.headers.get("set-cookie")).toContain("Max-Age=0");
      expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("callback", "success");
    },
  );

  it.each(origins)(
    "returns a generic error to $expectedOrigin without provider details",
    async ({ baseURL, expectedOrigin }) => {
      mocks.baseURL = baseURL;
      const response = await GET(
        new NextRequest("http://localhost:3000/auth/sso-callback?error=provider_secret_detail"),
      );

      expect(response.headers.get("location")).toBe(`${expectedOrigin}/auth/error?reason=callback`);
      expect(response.headers.get("location")).not.toContain("provider_secret_detail");
      expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("callback", "failure");
    },
  );

  it.each(origins)(
    "returns local callbacks to $expectedOrigin without AuthKit",
    async ({ baseURL, expectedOrigin }) => {
      mocks.authProvider = "local";
      mocks.baseURL = baseURL;

      const response = await GET(
        new NextRequest("http://localhost:3000/auth/sso-callback?code=local"),
      );

      expect(mocks.handleAuth).not.toHaveBeenCalled();
      expect(mocks.logManagedAuthOutcome).not.toHaveBeenCalled();
      expect(response.headers.get("location")).toBe(`${expectedOrigin}/apis`);
    },
  );
});
