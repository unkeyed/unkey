import { NextRequest } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  switchToOrg: vi.fn(),
  getAuth: vi.fn(),
  memberships: vi.fn(),
  findMany: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  switchToOrg: mocks.switchToOrg,
}));
vi.mock("@/lib/auth/get-auth", () => ({ getAuth: mocks.getAuth }));
vi.mock("@/lib/auth/server", () => ({ auth: { listMemberships: mocks.memberships } }));
vi.mock("@/lib/db", () => ({ db: { query: { workspaces: { findMany: mocks.findMany } } } }));

import { GET } from "./route";

describe("organization switch route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.switchToOrg.mockResolvedValue(undefined);
    mocks.getAuth.mockResolvedValue({ userId: "session-user" });
    mocks.memberships.mockResolvedValue({ data: [
      { status: "active", organization: { id: "org_123" } },
      { status: "inactive", organization: { id: "org_inactive" } },
      { status: "pending", organization: { id: "org_pending" } },
    ] });
    mocks.findMany.mockResolvedValue([{ orgId: "org_123", name: "Disabled workspace" }]);
  });

  it.each(["org_other_valid", "org_inactive", "org_pending"])(
    "rejects nonactive membership %s before querying workspace details or changing the session",
    async (orgId) => {
      const response = await GET(new NextRequest(`http://localhost:3000/auth/switch-organization?organization_id=${orgId}`));
      expect(response.headers.get("location")).toBe("http://localhost:3000/auth/error?reason=session");
      expect(response.headers.get("set-cookie")).toBeNull();
      expect(mocks.memberships).toHaveBeenCalledWith("session-user");
      expect(mocks.findMany).not.toHaveBeenCalled();
      expect(mocks.switchToOrg).not.toHaveBeenCalled();
    },
  );

  it("rejects missing or deleted workspace rows without switching or writing cookies", async () => {
    mocks.findMany.mockResolvedValue([]);
    const response = await GET(new NextRequest("http://localhost:3000/auth/switch-organization?organization_id=org_123"));
    expect(response.headers.get("location")).toBe("http://localhost:3000/auth/error?reason=session");
    expect(response.headers.get("set-cookie")).toBeNull();
    expect(mocks.switchToOrg).not.toHaveBeenCalled();
  });

  it("rejects signed-out requests without querying memberships or switching", async () => {
    mocks.getAuth.mockResolvedValue({ userId: null });
    const response = await GET(new NextRequest("http://localhost:3000/auth/switch-organization?organization_id=org_123"));
    expect(response.headers.get("location")).toBe("http://localhost:3000/auth/error?reason=session");
    expect(mocks.memberships).not.toHaveBeenCalled();
    expect(mocks.switchToOrg).not.toHaveBeenCalled();
  });

  it.each(["memberships", "findMany"] as const)("does not switch when %s fails", async (operation) => {
    mocks[operation].mockRejectedValue(new Error("private details"));
    const response = await GET(new NextRequest("http://localhost:3000/auth/switch-organization?organization_id=org_123"));
    expect(response.headers.get("location")).toBe("http://localhost:3000/auth/error?reason=session");
    expect(response.headers.get("set-cookie")).toBeNull();
    expect(mocks.switchToOrg).not.toHaveBeenCalled();
  });

  it("switches organizations before returning to a safe dashboard path", async () => {
    const response = await GET(
      new NextRequest(
        "http://localhost:3000/auth/switch-organization?organization_id=org_123&return_to=%2Facme%2Fapis",
      ),
    );

    expect(mocks.switchToOrg).toHaveBeenCalledWith("org_123");
    expect(response.headers.get("location")).toBe("http://localhost:3000/acme/apis");
  });

  it("falls back safely when the return path is external", async () => {
    const response = await GET(
      new NextRequest(
        "http://localhost:3000/auth/switch-organization?organization_id=org_123&return_to=https%3A%2F%2Fevil.example.com",
      ),
    );

    expect(response.headers.get("location")).toBe("http://localhost:3000/apis");
  });

  it("rejects malformed organization ids before switching", async () => {
    const response = await GET(
      new NextRequest(
        "http://localhost:3000/auth/switch-organization?organization_id=org_123%26next%3Devil",
      ),
    );

    expect(mocks.switchToOrg).not.toHaveBeenCalled();
    expect(response.headers.get("location")).toBe(
      "http://localhost:3000/auth/error?reason=session",
    );
  });

  it("lets AuthKit redirect signals propagate to the browser", async () => {
    const redirectSignal = new Error("NEXT_REDIRECT");
    mocks.switchToOrg.mockRejectedValue(redirectSignal);

    await expect(
      GET(
        new NextRequest("http://localhost:3000/auth/switch-organization?organization_id=org_123"),
      ),
    ).rejects.toBe(redirectSignal);
  });
});
