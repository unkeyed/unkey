import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getOrganizationMembership: vi.fn(),
  deactivateOrganizationMembership: vi.fn(),
  listOrganizationMemberships: vi.fn(),
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  getWorkOS: () => ({
    userManagement: {
      getOrganizationMembership: mocks.getOrganizationMembership,
      deactivateOrganizationMembership: mocks.deactivateOrganizationMembership,
      listOrganizationMemberships: mocks.listOrganizationMemberships,
    },
  }),
}));

import { WorkOSAuthProvider } from "../workos";

const membership = {
  id: "om_123",
  organizationId: "org_123",
};

describe("WorkOSAuthProvider server-side membership operations", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getOrganizationMembership.mockResolvedValue(membership);
    mocks.deactivateOrganizationMembership.mockResolvedValue(undefined);
  });

  it("does not expose custom interactive team administration", () => {
    const provider = new WorkOSAuthProvider();

    expect("inviteMember" in provider).toBe(false);
    expect("getInvitationList" in provider).toBe(false);
    expect("revokeOrgInvitation" in provider).toBe(false);
    expect("updateMembership" in provider).toBe(false);
    expect("removeMembership" in provider).toBe(false);
  });

  it("lists all active membership pages without returning a stale cursor", async () => {
    const provider = new WorkOSAuthProvider();
    vi.spyOn(provider, "getUser").mockResolvedValue({
      id: "user_123", email: "user@example.com", firstName: null, lastName: null,
      avatarUrl: null, fullName: null,
    });
    const member = (organizationId: string) => ({
      id: `mem_${organizationId}`, organizationId, organizationName: organizationId,
      role: { slug: "admin" }, status: "active", createdAt: "2026-01-01", updatedAt: "2026-01-01",
    });
    mocks.listOrganizationMemberships
      .mockResolvedValueOnce({ data: [member("org_first")], listMetadata: { after: "cursor_1" } })
      .mockResolvedValueOnce({ data: [member("org_second")], listMetadata: { after: "cursor_2" } })
      .mockResolvedValueOnce({ data: [member("org_last")], listMetadata: { after: null } });
    const result = await provider.listMemberships("user_123");
    expect(result.data.map((item) => item.organization.id)).toEqual(["org_first", "org_second", "org_last"]);
    expect(result.metadata).toEqual({});
    expect(mocks.listOrganizationMemberships.mock.calls).toEqual([
      [{ userId: "user_123", limit: 100, statuses: ["active"] }],
      [{ userId: "user_123", limit: 100, statuses: ["active"], after: "cursor_1" }],
      [{ userId: "user_123", limit: 100, statuses: ["active"], after: "cursor_2" }],
    ]);
  });

  it("does not deactivate a membership owned by another organization", async () => {
    mocks.getOrganizationMembership.mockResolvedValue({
      ...membership,
      organizationId: "org_other",
    });
    const provider = new WorkOSAuthProvider();

    await expect(provider.deactivateMembership("om_123", "org_123")).rejects.toMatchObject({
      name: "OrganizationScopeError",
    });
    expect(mocks.deactivateOrganizationMembership).not.toHaveBeenCalled();
  });

  it("retains scoped membership deactivation for server-side billing cleanup", async () => {
    const provider = new WorkOSAuthProvider();

    await expect(provider.deactivateMembership("om_123", "org_123")).resolves.toBeUndefined();

    expect(mocks.getOrganizationMembership).toHaveBeenCalledWith("om_123");
    expect(mocks.deactivateOrganizationMembership).toHaveBeenCalledWith("om_123");
  });
});
