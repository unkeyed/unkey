import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getInvitation: vi.fn(),
  getUser: vi.fn(),
  acceptInvitation: vi.fn(),
  switchOrg: vi.fn(),
  setSessionCookie: vi.fn(),
  setLastUsedOrgCookie: vi.fn(),
}));

vi.mock("@/lib/auth/server", () => ({
  auth: {
    getInvitation: mocks.getInvitation,
    getUser: mocks.getUser,
    acceptInvitation: mocks.acceptInvitation,
    switchOrg: mocks.switchOrg,
  },
}));

vi.mock("@/lib/auth/cookies", () => ({
  setSessionCookie: mocks.setSessionCookie,
  setLastUsedOrgCookie: mocks.setLastUsedOrgCookie,
}));

vi.mock("@/lib/auth/get-auth", () => ({ getAuth: vi.fn() }));

import { normalizeEmail, processPostAuthInvitation } from "./auth";

const USER_ID = "user_123";
const ORG_ID = "org_456";
const INVITATION_ID = "invitation_789";
const TOKEN = "tok_abc";

// Every user-facing failure must be one of these two literals. ENG-3014 was a
// raw provider error reaching the client.
const INVITATION_UNUSABLE = "This invitation is no longer valid. Ask for a new one.";
const EMAIL_MISMATCH = "This invitation was sent to a different email address.";

function givenInvitation(overrides: Record<string, unknown> = {}) {
  mocks.getInvitation.mockResolvedValue({
    id: INVITATION_ID,
    email: "invitee@example.com",
    state: "pending",
    organizationId: ORG_ID,
    ...overrides,
  });
}

function givenUser(email = "invitee@example.com") {
  mocks.getUser.mockResolvedValue({ id: USER_ID, email });
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(console, "error").mockImplementation(() => {});
  mocks.switchOrg.mockResolvedValue({ newToken: "sealed_token", expiresAt: new Date() });
});

describe("normalizeEmail", () => {
  it("ignores casing and surrounding whitespace", () => {
    expect(normalizeEmail("  User@Example.COM ")).toBe("user@example.com");
  });
});

describe("processPostAuthInvitation", () => {
  it("accepts the invitation and switches the user into the org", async () => {
    givenInvitation();
    givenUser();

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: true, organizationId: ORG_ID, switched: true });
    expect(mocks.acceptInvitation).toHaveBeenCalledWith(INVITATION_ID);
    expect(mocks.switchOrg).toHaveBeenCalledWith(ORG_ID);
    expect(mocks.setSessionCookie).toHaveBeenCalled();
  });

  /**
   * Guarantees an invitation is only ever consumed by the account it was
   * addressed to. acceptInvitation is irreversible, so a mismatched account
   * must be rejected before it runs, not after.
   */
  it("never consumes an invitation addressed to a different account", async () => {
    givenInvitation({ email: "invitee@example.com" });
    givenUser("someone.else@example.com");

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: false, error: EMAIL_MISMATCH });
    expect(mocks.acceptInvitation).not.toHaveBeenCalled();
    expect(mocks.switchOrg).not.toHaveBeenCalled();
  });

  /**
   * The invited address and the stored account email routinely differ in
   * casing. Comparing them raw locked legitimate invitees out of the org.
   */
  it("matches the invited address regardless of casing or padding", async () => {
    givenInvitation({ email: " Invitee@Example.COM " });
    givenUser("invitee@example.com");

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: true, organizationId: ORG_ID, switched: true });
    expect(mocks.acceptInvitation).toHaveBeenCalledWith(INVITATION_ID);
  });

  /**
   * Guarantees a stale invitation link is inert. Re-accepting an already
   * consumed invitation would silently move an active user out of the org they
   * are currently working in.
   */
  it("does nothing for an invitation that is not pending", async () => {
    givenInvitation({ state: "accepted" });
    givenUser();

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: false, error: INVITATION_UNUSABLE });
    expect(mocks.acceptInvitation).not.toHaveBeenCalled();
    expect(mocks.switchOrg).not.toHaveBeenCalled();
  });

  /**
   * Guarantees the endpoint is not an invitation-token oracle: an unknown
   * token and a real-but-unusable one must be indistinguishable to the caller,
   * or any signed-in user could probe tokens for validity.
   */
  it("reports unknown and revoked invitations identically", async () => {
    mocks.getInvitation.mockResolvedValue(null);
    givenUser();
    const unknown = await processPostAuthInvitation(TOKEN, USER_ID);

    vi.clearAllMocks();
    givenInvitation({ state: "revoked" });
    givenUser();
    const revoked = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(unknown).toEqual({ success: false, error: INVITATION_UNUSABLE });
    expect(revoked).toEqual(unknown);
  });

  /**
   * Guarantees ENG-3014 stays fixed: a provider failure must surface as a
   * fixed, user-safe literal, never as the provider's own error text.
   */
  it("does not leak provider error text to the caller", async () => {
    mocks.getInvitation.mockRejectedValue(new Error("WorkOS 500: internal_error at shard 7"));

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: false, error: INVITATION_UNUSABLE });
  });

  /**
   * Guarantees a failed org switch is not reported as a plain failure. The
   * invitation is already consumed at that point, so the user IS a member;
   * telling them it failed would strand them with nothing left to retry.
   */
  it("reports partial success when the org switch fails after acceptance", async () => {
    givenInvitation();
    givenUser();
    mocks.switchOrg.mockRejectedValue(new Error("Organization switch failed mfa_enrollment"));

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: true, organizationId: ORG_ID, switched: false });
    expect(mocks.acceptInvitation).toHaveBeenCalledWith(INVITATION_ID);
  });

  /**
   * The last-used-org cookie only preselects a workspace on the next sign-in.
   * A completed switch must not fail because that cookie could not be written.
   */
  it("still succeeds when the last-used-org cookie cannot be written", async () => {
    givenInvitation();
    givenUser();
    mocks.setLastUsedOrgCookie.mockRejectedValue(new Error("cookies() not available"));

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: true, organizationId: ORG_ID, switched: true });
  });

  it("rejects an invitation carrying no organization", async () => {
    givenInvitation({ organizationId: null });
    givenUser();

    const result = await processPostAuthInvitation(TOKEN, USER_ID);

    expect(result).toEqual({ success: false, error: INVITATION_UNUSABLE });
    expect(mocks.acceptInvitation).not.toHaveBeenCalled();
  });
});
