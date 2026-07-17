import { NextRequest } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getAuth: vi.fn(),
  processPostAuthInvitation: vi.fn(),
}));

vi.mock("@/lib/auth/get-auth", () => ({ getAuth: mocks.getAuth }));
vi.mock("@/lib/auth", () => ({ processPostAuthInvitation: mocks.processPostAuthInvitation }));

import { POST } from "./route";

const USER_ID = "user_123";
const ORG_ID = "org_456";

function postRequest(body: string): NextRequest {
  return new NextRequest("https://app.unkey.com/api/auth/invitation", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.getAuth.mockResolvedValue({ userId: USER_ID });
});

describe("POST /api/auth/invitation", () => {
  it("accepts the invitation and reports the org it switched into", async () => {
    mocks.processPostAuthInvitation.mockResolvedValue({
      success: true,
      organizationId: ORG_ID,
      switched: true,
    });

    const response = await POST(postRequest(JSON.stringify({ invitationToken: "tok_abc" })));

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      success: true,
      organizationId: ORG_ID,
      switched: true,
    });
    expect(mocks.processPostAuthInvitation).toHaveBeenCalledWith("tok_abc", USER_ID);
  });

  /**
   * Guarantees the caller can tell "you joined but we could not open the
   * workspace" apart from a clean success. The client needs it to explain the
   * outcome instead of reloading into the wrong workspace.
   */
  it("passes the partial-success flag through on a failed org switch", async () => {
    mocks.processPostAuthInvitation.mockResolvedValue({
      success: true,
      organizationId: ORG_ID,
      switched: false,
    });

    const response = await POST(postRequest(JSON.stringify({ invitationToken: "tok_abc" })));

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({ success: true, switched: false });
  });

  /**
   * Guarantees ENG-3014 stays fixed at the boundary that regressed: the client
   * receives the vetted message the domain layer chose, and a malformed body is
   * a client error, not a 500 that pages an operator.
   */
  it("returns the domain layer's user-safe message on failure", async () => {
    mocks.processPostAuthInvitation.mockResolvedValue({
      success: false,
      error: "This invitation was sent to a different email address.",
    });

    const response = await POST(postRequest(JSON.stringify({ invitationToken: "tok_abc" })));

    expect(response.status).toBe(400);
    await expect(response.json()).resolves.toEqual({
      success: false,
      error: "This invitation was sent to a different email address.",
    });
  });

  it("rejects a malformed body as a client error", async () => {
    const response = await POST(postRequest("not json{"));

    expect(response.status).toBe(400);
    await expect(response.json()).resolves.toEqual({
      success: false,
      error: "Invalid JSON body",
    });
    expect(mocks.processPostAuthInvitation).not.toHaveBeenCalled();
  });

  it("rejects a body carrying no invitation token", async () => {
    const response = await POST(postRequest(JSON.stringify({ invitationToken: "   " })));

    expect(response.status).toBe(400);
    expect(mocks.processPostAuthInvitation).not.toHaveBeenCalled();
  });

  /**
   * Guarantees an unauthenticated caller can never reach the acceptance path,
   * which irreversibly consumes the invitation.
   */
  it("refuses to process an invitation without a session", async () => {
    mocks.getAuth.mockResolvedValue({ userId: null });

    const response = await POST(postRequest(JSON.stringify({ invitationToken: "tok_abc" })));

    expect(response.status).toBe(401);
    expect(mocks.processPostAuthInvitation).not.toHaveBeenCalled();
  });
});
