import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getOrganizationMembership: vi.fn(),
  deactivateOrganizationMembership: vi.fn(),
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  getWorkOS: () => ({
    userManagement: {
      getOrganizationMembership: mocks.getOrganizationMembership,
      deactivateOrganizationMembership: mocks.deactivateOrganizationMembership,
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
