// @vitest-environment node

import { NextRequest } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/auth/get-auth", () => ({
  getAuth: vi.fn(),
}));

vi.mock("@/lib/auth/server", () => ({
  auth: {
    getUser: vi.fn(),
  },
}));

vi.mock("@/lib/env", () => ({
  env: vi.fn(),
}));

import { getAuth } from "@/lib/auth/get-auth";
import { auth as authProvider } from "@/lib/auth/server";
import { LOCAL_AUTH_PERMISSIONS } from "@/lib/auth/types";
import { env } from "@/lib/env";
import { jwtVerify } from "jose";
import { POST } from "./route";

const mockedGetAuth = vi.mocked(getAuth);
const mockedAuthProvider = vi.mocked(authProvider);
const mockedEnv = vi.mocked(env);

function makeRequest(headers: Record<string, string> = {}): NextRequest {
  return new NextRequest("http://localhost/proxy/v2/apis.listKeys", {
    method: "POST",
    headers,
    body: JSON.stringify({}),
  });
}

const params = Promise.resolve({ path: ["v2", "apis.listKeys"] });
const encoder = new TextEncoder();

describe("dashboard proxy POST", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ ok: true }), { status: 200 })),
    );
    mockedEnv.mockReturnValue({
      UNKEY_API_URL: "https://api.example.test",
      UNKEY_JWT_SECRET: "test-secret-with-at-least-32-bytes-of-entropy",
    } as ReturnType<typeof env>);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects requests that smuggle an Authorization header", async () => {
    // The proxy mints its own JWT; accepting a caller-supplied Authorization
    // header would let a browser forward an attacker-controlled bearer to the
    // upstream API under the dashboard user's identity.
    const res = await POST(makeRequest({ authorization: "Bearer attacker" }), {
      params,
    });

    expect(res.status).toBe(400);
    expect(mockedGetAuth).not.toHaveBeenCalled();
  });

  it("returns 401 when the caller is unauthenticated", async () => {
    // Anonymous requests must be rejected before any workspace lookup so an
    // unauthenticated caller cannot probe org-to-workspace mappings.
    mockedGetAuth.mockResolvedValue({ userId: null, orgId: null, role: null });

    const res = await POST(makeRequest(), { params });

    expect(res.status).toBe(401);
  });

  it("forwards the WorkOS access token when present", async () => {
    mockedGetAuth.mockResolvedValue({
      userId: "user_1",
      orgId: "org_1",
      accessToken: "workos_access_token",
      role: "owner",
    });

    const res = await POST(makeRequest({ accept: "application/json" }), { params });

    expect(res.status).toBe(200);
    expect(mockedAuthProvider.getUser).not.toHaveBeenCalled();
    expect(fetch).toHaveBeenCalledOnce();
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(init).toBeDefined();
    const headers = new Headers(init?.headers);
    expect(headers.get("authorization")).toBe("Bearer workos_access_token");
  });

  it("mints a fallback proxy JWT scoped by org id", async () => {
    mockedGetAuth.mockResolvedValue({
      userId: "user_1",
      orgId: "org_1",
      permissions: LOCAL_AUTH_PERMISSIONS,
      role: "owner",
    });
    mockedAuthProvider.getUser.mockResolvedValue({
      fullName: "Test User",
      email: "test@example.test",
    } as Awaited<ReturnType<typeof authProvider.getUser>>);

    const res = await POST(makeRequest({ accept: "application/json" }), { params });

    expect(res.status).toBe(200);
    expect(fetch).toHaveBeenCalledOnce();
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(init).toBeDefined();
    const headers = new Headers(init?.headers);
    const authorization = headers.get("authorization");
    expect(authorization).toMatch(/^Bearer /);

    const token = authorization?.replace("Bearer ", "");
    expect(token).toBeDefined();
    const { payload } = await jwtVerify(
      token ?? "",
      encoder.encode("test-secret-with-at-least-32-bytes-of-entropy"),
      {
        issuer: "app.unkey.com",
        audience: "api.unkey.com",
        subject: "user_1",
      },
    );
    expect(payload.org).toEqual({ id: "org_1" });
    expect(payload["org.id"]).toBeUndefined();
    expect(payload.wid).toBeUndefined();
    expect(payload.name).toBe("Test User");
    expect(payload.perms).toEqual(LOCAL_AUTH_PERMISSIONS);
  });
});
