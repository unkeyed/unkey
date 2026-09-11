import { NextRequest, NextResponse } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authProvider: "workos" as "workos" | "local",
  handleLocalJoin: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider }),
}));

vi.mock("./local-join", () => ({
  handleLocalJoin: mocks.handleLocalJoin,
}));

import { GET } from "./route";

const TOKEN = "Z1uX3RbwcIl5fIGJJJCXXisdI";

describe("GET /join", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authProvider = "workos";
  });

  it("does not run a custom invitation flow in WorkOS mode", async () => {
    const response = await GET(
      new NextRequest(`http://localhost:3000/join?invitation_token=${TOKEN}`),
    );

    expect(response.status).toBe(404);
    expect(mocks.handleLocalJoin).not.toHaveBeenCalled();
  });

  it("preserves local join behavior", async () => {
    mocks.authProvider = "local";
    mocks.handleLocalJoin.mockResolvedValue(NextResponse.redirect("http://localhost:3000/apis"));
    const request = new NextRequest(`http://localhost:3000/join?invitation_token=${TOKEN}`);

    const response = await GET(request);

    expect(mocks.handleLocalJoin).toHaveBeenCalledWith(request);
    expect(response.headers.get("location")).toBe("http://localhost:3000/apis");
  });
});
