import { NextRequest } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  switchToOrg: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  switchToOrg: mocks.switchToOrg,
}));

import { GET } from "./route";

describe("organization switch route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.switchToOrg.mockResolvedValue(undefined);
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
