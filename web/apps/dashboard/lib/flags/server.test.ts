// @vitest-environment node

import { getAuth } from "@/lib/auth/get-auth";
import type { V2WorkspaceGetFlagsResponseBody } from "@unkey/api/models/components";
import { jwtVerify } from "jose";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { loadFlags } from "./server";

vi.mock("@/lib/auth/get-auth", () => ({ getAuth: vi.fn() }));

vi.mock("@/lib/env", () => ({
  env: () => ({
    UNKEY_API_URL: "https://api.example.test",
    UNKEY_JWT_SECRET: "test-secret-with-at-least-32-bytes-of-entropy",
  }),
}));

beforeEach(() => vi.mocked(getAuth).mockReset());
afterEach(() => vi.unstubAllGlobals());

it("uses a workspace-scoped signed token for local auth", async () => {
  const userId = crypto.randomUUID();
  const orgId = crypto.randomUUID();
  vi.stubGlobal("fetch", async (request: Request) => {
    const token = request.headers.get("authorization")?.replace(/^Bearer /, "") ?? "";
    const { payload } = await jwtVerify(
      token,
      new TextEncoder().encode("test-secret-with-at-least-32-bytes-of-entropy"),
      { issuer: "app.unkey.com", audience: "api.unkey.com", subject: userId },
    );
    expect(payload.org).toEqual({ id: orgId });
    expect(payload.roles).toEqual(["admin"]);
    return Response.json({ meta: { requestId: crypto.randomUUID() }, data: {} });
  });
  vi.mocked(getAuth).mockResolvedValue({ userId, orgId, role: "admin" });
  await expect(loadFlags()).resolves.toEqual({});
});

it("does not request flags without an authenticated workspace", async () => {
  const fetcher = vi.fn();
  vi.stubGlobal("fetch", fetcher);
  vi.mocked(getAuth).mockResolvedValue({ userId: null, orgId: null, role: null });
  await expect(loadFlags()).resolves.toBeNull();
  vi.mocked(getAuth).mockResolvedValue({ userId: crypto.randomUUID(), orgId: null, role: null });
  await expect(loadFlags()).resolves.toBeNull();
  expect(fetcher).not.toHaveBeenCalled();
});

it("loads server flags directly with the session credential and preserves disabled values", async () => {
  const data: V2WorkspaceGetFlagsResponseBody["data"] = {
    deploy_billing: {
      enabled: false,
      description: "Billing",
      allowOptIn: true,
      allowOptOut: false,
    },
  };
  vi.stubGlobal("fetch", async (request: Request) => {
    expect(request.url).toBe("https://api.example.test/v2/workspace.getFlags");
    expect(request.method).toBe("POST");
    expect(request.cache).toBe("no-store");
    expect(request.redirect).toBe("error");
    expect(request.headers.get("authorization")).toBe("Bearer session-token");
    return Response.json({ meta: { requestId: crypto.randomUUID() }, data });
  });
  vi.mocked(getAuth).mockResolvedValue({
    userId: crypto.randomUUID(),
    orgId: crypto.randomUUID(),
    role: "viewer",
    accessToken: "session-token",
  });
  await expect(loadFlags()).resolves.toEqual(data);
});
