import { NextRequest } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/auth/get-auth", () => ({
  getAuth: vi.fn(),
}));

vi.mock("@/lib/auth/server", () => ({
  auth: {
    getUser: vi.fn(),
  },
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      workspaces: {
        findFirst: vi.fn(),
      },
    },
  },
}));

vi.mock("@/lib/env", () => ({
  env: vi.fn(),
}));

import { getAuth } from "@/lib/auth/get-auth";
import { auth as authProvider } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { POST } from "./route";

const mockedGetAuth = vi.mocked(getAuth);
const mockedAuthProvider = vi.mocked(authProvider);
const mockedFindFirst = vi.mocked(db.query.workspaces.findFirst);
const mockedEnv = vi.mocked(env);

function makeRequest(headers: Record<string, string> = {}): NextRequest {
  return new NextRequest("http://localhost/proxy/v2/apis.listKeys", {
    method: "POST",
    headers,
    body: JSON.stringify({}),
  });
}

const params = Promise.resolve({ path: ["v2", "apis.listKeys"] });

describe("dashboard proxy POST", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedEnv.mockReturnValue({
      UNKEY_API_URL: "https://api.example.test",
      UNKEY_JWT_SECRET: "test-secret-with-at-least-32-bytes-of-entropy",
    } as ReturnType<typeof env>);
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
    expect(mockedFindFirst).not.toHaveBeenCalled();
  });

  it("returns 404 when the authenticated org has no workspace", async () => {
    // A signed-in user whose org has no workspace must not be issued a JWT, or
    // they would acquire a token with an empty workspace_id and reach the API
    // with no workspace scope.
    mockedGetAuth.mockResolvedValue({ userId: "user_1", orgId: "org_1", role: "owner" });
    mockedAuthProvider.getUser.mockResolvedValue({
      fullName: "Test User",
      email: "test@example.test",
    } as Awaited<ReturnType<typeof authProvider.getUser>>);
    mockedFindFirst.mockResolvedValue(undefined);

    const res = await POST(makeRequest(), { params });

    expect(res.status).toBe(404);
  });
});
