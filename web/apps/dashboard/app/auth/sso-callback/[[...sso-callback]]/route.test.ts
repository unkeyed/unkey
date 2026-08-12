import { NextRequest, NextResponse } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authProvider: "workos" as "workos" | "local",
  handleAuth: vi.fn(),
  logManagedAuthOutcome: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider }),
  workosAuthEnv: vi.fn(),
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  handleAuth: mocks.handleAuth,
}));

vi.mock("@/lib/auth/telemetry", () => ({
  logManagedAuthOutcome: mocks.logManagedAuthOutcome,
}));

import { GET } from "./route";

describe("AuthKit callback", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authProvider = "workos";
    mocks.handleAuth.mockImplementation(
      (options: {
        onSuccess: () => void | Promise<void>;
        onError: (params: { request: NextRequest }) => Response | Promise<Response>;
      }) =>
        async (request: NextRequest) => {
          if (request.nextUrl.searchParams.has("error")) {
            return options.onError({ request });
          }
          await options.onSuccess();
          return NextResponse.redirect(new URL("/apis", request.url));
        },
    );
  });

  it("delegates the WorkOS callback transaction to AuthKit and clears the legacy cookie", async () => {
    const response = await GET(
      new NextRequest("http://localhost:3000/auth/sso-callback?code=code&state=state", {
        headers: { cookie: "unkey-session=legacy" },
      }),
    );

    expect(mocks.handleAuth).toHaveBeenCalledOnce();
    expect(response.headers.get("location")).toBe("http://localhost:3000/apis");
    expect(response.headers.get("set-cookie")).toContain("unkey-session=");
    expect(response.headers.get("set-cookie")).toContain("Max-Age=0");
    expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("callback", "success");
  });

  it("uses a generic terminal error without exposing provider details", async () => {
    const response = await GET(
      new NextRequest("http://localhost:3000/auth/sso-callback?error=provider_secret_detail"),
    );

    expect(response.headers.get("location")).toBe(
      "http://localhost:3000/auth/error?reason=callback",
    );
    expect(response.headers.get("location")).not.toContain("provider_secret_detail");
    expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("callback", "failure");
  });

  it("preserves the local callback implementation without invoking AuthKit", async () => {
    mocks.authProvider = "local";

    const response = await GET(
      new NextRequest("http://localhost:3000/auth/sso-callback?code=local"),
    );

    expect(mocks.handleAuth).not.toHaveBeenCalled();
    expect(mocks.logManagedAuthOutcome).not.toHaveBeenCalled();
    expect(response.headers.get("location")).toBe("http://localhost:3000/apis");
  });
});
