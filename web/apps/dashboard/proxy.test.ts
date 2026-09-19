import { unstable_doesMiddlewareMatch } from "next/experimental/testing/server";
import { NextRequest, NextResponse } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type AuthkitOptions = {
  redirectUri?: string;
  screenHint: "sign-up" | "sign-in";
  onSessionRefreshSuccess: () => void;
  onSessionRefreshError: () => void;
};

type AuthkitResult = {
  session: {
    user: { id: string } | null;
  };
  headers: Headers;
  authorizationUrl?: string;
};

type HeaderOptions = {
  redirect?: string;
};

const mocks = vi.hoisted(() => ({
  authProvider: "workos" as "workos" | "local",
  dashboardBaseUrl: "http://localhost:3000",
  vercelUrl: undefined as string | undefined,
  authkit: vi.fn<(request: NextRequest, options: AuthkitOptions) => Promise<AuthkitResult>>(),
  handleAuthkitHeaders:
    vi.fn<(request: NextRequest, headers: Headers, options?: HeaderOptions) => NextResponse>(),
  logManagedAuthOutcome: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider, VERCEL_URL: mocks.vercelUrl }),
  workosAuthEnv: vi.fn(),
}));

vi.mock("@/lib/auth/telemetry", () => ({
  logManagedAuthOutcome: mocks.logManagedAuthOutcome,
}));

vi.mock("@/lib/utils", () => ({
  getBaseUrl: () => mocks.dashboardBaseUrl,
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  authkit: mocks.authkit,
  handleAuthkitHeaders: mocks.handleAuthkitHeaders,
}));

import proxy, { config } from "./proxy";

function currentAuthkitOptions(): AuthkitOptions {
  const options = mocks.authkit.mock.calls[0]?.[1];
  if (!options) {
    throw new Error("AuthKit was not called");
  }
  return options;
}

function authenticatedResponse(): AuthkitResult {
  return { session: { user: { id: "user_1" } }, headers: new Headers() };
}

function heldRefresh() {
  let release: () => void = () => {};
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  return { gate, release };
}

async function settledTick(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 20));
}

describe("proxy auth mode split", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authProvider = "workos";
    mocks.dashboardBaseUrl = "http://localhost:3000";
    mocks.vercelUrl = undefined;
    mocks.authkit.mockResolvedValue({
      session: { user: null },
      headers: new Headers(),
      authorizationUrl: "https://authkit.example.com/authorize",
    });
    mocks.handleAuthkitHeaders.mockImplementation((_request, _headers, options) =>
      options?.redirect ? NextResponse.redirect(options.redirect) : NextResponse.next(),
    );
  });

  it("does not invoke AuthKit in local mode", async () => {
    mocks.authProvider = "local";

    const response = await proxy(new NextRequest("http://localhost:3000/apis"));

    expect(response.status).toBe(200);
    expect(mocks.authkit).not.toHaveBeenCalled();
    expect(mocks.handleAuthkitHeaders).not.toHaveBeenCalled();
  });

  it("preserves the local invitation compatibility redirect", async () => {
    mocks.authProvider = "local";

    const response = await proxy(
      new NextRequest("http://localhost:3000/auth/join?invitation_token=local-token"),
    );

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe(
      "http://localhost:3000/join?invitation_token=local-token",
    );
    expect(mocks.authkit).not.toHaveBeenCalled();
  });

  it.each([
    "/api/trpc/user.getCurrentUser,workspace.getCurrent",
    "/proxy/v2/keys.updateKey",
    "/proxy/v2/apis.listKeys",
  ])("runs AuthKit on dotted API path %s", (path) => {
    expect(
      unstable_doesMiddlewareMatch({
        config,
        url: `http://localhost:3000${path}`,
      }),
    ).toBe(true);
  });

  it("does not match public static files", () => {
    expect(
      unstable_doesMiddlewareMatch({
        config,
        url: "http://localhost:3000/logo.svg",
      }),
    ).toBe(false);
  });

  it.each(["/monitoring", "/monitoring?o=123&p=456&r=us"])(
    "does not redirect the Sentry tunnel %s through authentication",
    (path) => {
      expect(
        unstable_doesMiddlewareMatch({
          config,
          url: `http://localhost:3000${path}`,
        }),
      ).toBe(false);
    },
  );

  it.each(["/monitoring-workspace", "/monitoring/apis"])(
    "still authenticates dashboard paths starting with monitoring: %s",
    (path) => {
      expect(
        unstable_doesMiddlewareMatch({
          config,
          url: `http://localhost:3000${path}`,
        }),
      ).toBe(true);
    },
  );

  it("does not create an unused AuthKit transaction for public route handlers", async () => {
    const response = await proxy(new NextRequest("http://localhost:3000/api/webhooks/workos"));

    expect(response.status).toBe(200);
    expect(mocks.authkit).not.toHaveBeenCalled();
    expect(mocks.handleAuthkitHeaders).not.toHaveBeenCalled();
  });

  it("lets the retired WorkOS join handler return 404 without starting AuthKit", async () => {
    const response = await proxy(
      new NextRequest("http://localhost:3000/join?invitation_token=token"),
    );

    expect(response.status).toBe(200);
    expect(mocks.authkit).not.toHaveBeenCalled();
    expect(mocks.handleAuthkitHeaders).not.toHaveBeenCalled();
  });

  it("still supplies AuthKit headers to public pages rendered by the root provider", async () => {
    const response = await proxy(new NextRequest("http://localhost:3000/auth/error"));

    expect(response.status).toBe(200);
    expect(mocks.authkit).toHaveBeenCalledOnce();
    expect(mocks.handleAuthkitHeaders).toHaveBeenCalledOnce();
  });

  it("starts hosted sign-in with a sanitized deep link", async () => {
    await proxy(
      new NextRequest("http://localhost:3000/auth/sign-in?redirect=%2Fworkspace%2Fsettings%2Fteam"),
    );

    const authkitRequest = mocks.authkit.mock.calls[0]?.[0];
    expect(authkitRequest).toBeInstanceOf(NextRequest);
    expect(authkitRequest?.nextUrl.pathname).toBe("/workspace/settings/team");
    expect(mocks.handleAuthkitHeaders).toHaveBeenCalledWith(
      expect.any(NextRequest),
      expect.any(Headers),
      { redirect: "https://authkit.example.com/authorize" },
    );
  });

  it("uses the current Vercel deployment for the WorkOS callback", async () => {
    mocks.dashboardBaseUrl = "https://unkey-git-auth-no-custom.vercel.app";
    mocks.vercelUrl = "unkey-dpl_123.unkey.vercel.app";

    await proxy(new NextRequest("https://unkey-dpl_123.unkey.vercel.app/apis"));

    expect(currentAuthkitOptions().redirectUri).toBe(
      "https://unkey-dpl_123.unkey.vercel.app/auth/sso-callback",
    );
  });

  it("falls back to the configured callback origin for an untrusted host", async () => {
    mocks.dashboardBaseUrl = "https://unkey-git-auth-no-custom.vercel.app";

    await proxy(new NextRequest("https://untrusted.example.com/apis"));

    expect(currentAuthkitOptions().redirectUri).toBe(
      "https://unkey-git-auth-no-custom.vercel.app/auth/sso-callback",
    );
  });

  it("falls back safely when the requested deep link is unsafe", async () => {
    await proxy(
      new NextRequest("http://localhost:3000/auth/sign-in?redirect=https%3A%2F%2Fevil.example.com"),
    );

    const authkitRequest = mocks.authkit.mock.calls[0]?.[0];
    expect(authkitRequest).toBeInstanceOf(NextRequest);
    expect(authkitRequest?.nextUrl.pathname).toBe("/apis");
  });

  it("redirects a signed-out protected page and clears a legacy session cookie", async () => {
    const response = await proxy(
      new NextRequest("http://localhost:3000/apis", {
        headers: { cookie: "unkey-session=legacy" },
      }),
    );

    expect(response.headers.get("location")).toBe("https://authkit.example.com/authorize");
    expect(response.headers.get("set-cookie")).toContain("unkey-session=");
    expect(response.headers.get("set-cookie")).toContain("Max-Age=0");
  });

  it.each(["/api/example", "/proxy/v2/keys.updateKey"])(
    "passes a signed-out request to %s through without an interactive redirect",
    async (path) => {
      const request = new NextRequest(`http://localhost:3000${path}`, { method: "POST" });
      const response = await proxy(request);

      expect(response.status).toBe(200);
      expect(response.headers.get("location")).toBeNull();
      expect(mocks.authkit).toHaveBeenCalledWith(request, expect.any(Object));
      expect(mocks.handleAuthkitHeaders).toHaveBeenCalledWith(request, expect.any(Headers));
    },
  );

  it("uses the AuthKit response helper on an authenticated request", async () => {
    mocks.authkit.mockResolvedValue({
      session: { user: { id: "user_123" } },
      headers: new Headers({ "x-workos-session": "trusted" }),
      authorizationUrl: "https://authkit.example.com/authorize",
    });

    await proxy(
      new NextRequest("http://localhost:3000/apis", {
        headers: { "x-workos-session": "attacker-controlled" },
      }),
    );

    expect(mocks.handleAuthkitHeaders).toHaveBeenCalledWith(
      expect.any(NextRequest),
      expect.any(Headers),
    );
  });

  it("records fixed session-refresh outcomes without provider details", async () => {
    await proxy(new NextRequest("http://localhost:3000/auth/error"));
    const options = currentAuthkitOptions();

    options.onSessionRefreshSuccess();
    options.onSessionRefreshError();

    expect(mocks.logManagedAuthOutcome).toHaveBeenNthCalledWith(1, "session_refresh", "success");
    expect(mocks.logManagedAuthOutcome).toHaveBeenNthCalledWith(2, "session_refresh", "failure");
  });

  it("shares one session refresh between concurrent requests with the same session", async () => {
    const { gate, release } = heldRefresh();
    mocks.authkit.mockImplementation(async () => {
      await gate;
      return authenticatedResponse();
    });

    const request = () =>
      new NextRequest("http://localhost:3000/apis", {
        headers: { cookie: "wos-session=sealed-shared" },
      });
    const first = proxy(request());
    await vi.waitFor(() => expect(mocks.authkit).toHaveBeenCalledOnce());
    const second = proxy(request());
    await settledTick();
    release();

    await Promise.all([first, second]);
    expect(mocks.authkit).toHaveBeenCalledOnce();
    expect(mocks.handleAuthkitHeaders).toHaveBeenCalledTimes(2);
  });

  it("refreshes separately for requests carrying different sessions", async () => {
    mocks.authkit.mockResolvedValue(authenticatedResponse());

    await proxy(
      new NextRequest("http://localhost:3000/apis", {
        headers: { cookie: "wos-session=sealed-one" },
      }),
    );
    await proxy(
      new NextRequest("http://localhost:3000/apis", {
        headers: { cookie: "wos-session=sealed-two" },
      }),
    );

    expect(mocks.authkit).toHaveBeenCalledTimes(2);
  });

  it("does not share a refresh between sign-in entry requests", async () => {
    const { gate, release } = heldRefresh();
    mocks.authkit.mockImplementation(async () => {
      await gate;
      return {
        session: { user: null },
        headers: new Headers(),
        authorizationUrl: "https://authkit.example.com/authorize",
      };
    });

    const entry = (redirect: string) =>
      new NextRequest(`http://localhost:3000/auth/sign-in?redirect=${redirect}`, {
        headers: { cookie: "wos-session=sealed-entry" },
      });
    const first = proxy(entry("%2Fapis"));
    await vi.waitFor(() => expect(mocks.authkit).toHaveBeenCalledOnce());
    const second = proxy(entry("%2Fprojects"));
    await settledTick();
    release();

    await Promise.all([first, second]);
    expect(mocks.authkit).toHaveBeenCalledTimes(2);
  });

  it("gives each shared request its own return path", async () => {
    const returnPaths: (string | null)[] = [];
    const { gate, release } = heldRefresh();
    mocks.authkit.mockImplementation(async () => {
      await gate;
      return authenticatedResponse();
    });
    mocks.handleAuthkitHeaders.mockImplementation((_request, headers, options) => {
      returnPaths.push(headers.get("x-url"));
      return options?.redirect ? NextResponse.redirect(options.redirect) : NextResponse.next();
    });

    const request = (path: string) =>
      new NextRequest(`http://localhost:3000${path}`, {
        headers: { cookie: "wos-session=sealed-paths" },
      });
    const first = proxy(request("/apis"));
    await vi.waitFor(() => expect(mocks.authkit).toHaveBeenCalledOnce());
    const second = proxy(request("/projects"));
    await settledTick();
    release();

    await Promise.all([first, second]);
    expect(mocks.authkit).toHaveBeenCalledOnce();
    expect(returnPaths.toSorted()).toEqual([
      "http://localhost:3000/apis",
      "http://localhost:3000/projects",
    ]);
  });
});
