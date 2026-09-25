import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getOrganizationMembership: vi.fn(),
  deactivateOrganizationMembership: vi.fn(),
  listOrganizationMemberships: vi.fn(),
  listUsers: vi.fn(),
  getUser: vi.fn(),
  getOrganization: vi.fn(),
  logOperation: vi.fn(),
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  getWorkOS: () => ({
    organizations: {
      getOrganization: mocks.getOrganization,
    },
    userManagement: {
      getOrganizationMembership: mocks.getOrganizationMembership,
      deactivateOrganizationMembership: mocks.deactivateOrganizationMembership,
      listOrganizationMemberships: mocks.listOrganizationMemberships,
      listUsers: mocks.listUsers,
      getUser: mocks.getUser,
    },
  }),
}));

vi.mock("@/lib/logging", () => ({ logOperation: mocks.logOperation }));

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
      id: "user_123",
      email: "user@example.com",
      firstName: null,
      lastName: null,
      avatarUrl: null,
      fullName: null,
    });
    const member = (organizationId: string) => ({
      id: `mem_${organizationId}`,
      organizationId,
      organizationName: organizationId,
      role: { slug: "admin" },
      status: "active",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    mocks.listOrganizationMemberships
      .mockResolvedValueOnce({ data: [member("org_first")], listMetadata: { after: "cursor_1" } })
      .mockResolvedValueOnce({ data: [member("org_second")], listMetadata: { after: "cursor_2" } })
      .mockResolvedValueOnce({ data: [member("org_last")], listMetadata: { after: null } });
    const result = await provider.listMemberships("user_123");
    expect(result.data.map((item) => item.organization.id)).toEqual([
      "org_first",
      "org_second",
      "org_last",
    ]);
    expect(result.metadata).toEqual({});
    expect(mocks.listOrganizationMemberships.mock.calls.map(([call]) => call.after)).toEqual([
      undefined,
      "cursor_1",
      "cursor_2",
    ]);
  });

  it("filters memberships at the provider when an organization is given", async () => {
    const provider = new WorkOSAuthProvider();
    vi.spyOn(provider, "getUser").mockResolvedValue({
      id: "user_123",
      email: "user@example.com",
      firstName: null,
      lastName: null,
      avatarUrl: null,
      fullName: null,
    });
    mocks.listOrganizationMemberships.mockResolvedValue({ data: [], listMetadata: {} });

    await provider.listMemberships("user_123", "org_123");

    expect(mocks.listOrganizationMemberships).toHaveBeenCalledOnce();
    expect(mocks.listOrganizationMemberships.mock.calls[0][0]).toMatchObject({
      userId: "user_123",
      organizationId: "org_123",
    });
  });

  it("lists active organization ids without fetching the user", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.listOrganizationMemberships.mockResolvedValue({
      data: [
        { organizationId: "org_active", status: "active" },
        { organizationId: "org_inactive", status: "inactive" },
      ],
      listMetadata: {},
    });

    await expect(provider.listActiveOrganizationIds("user_123", "org_active")).resolves.toEqual([
      "org_active",
    ]);
    expect(mocks.getUser).not.toHaveBeenCalled();
    expect(mocks.listOrganizationMemberships.mock.calls[0][0]).toMatchObject({
      userId: "user_123",
      organizationId: "org_active",
      statuses: ["active"],
    });
  });

  it("reports a provider outage instead of reporting the user as missing", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getUser.mockRejectedValue(Object.assign(new Error("upstream down"), { status: 503 }));

    await expect(provider.getUser("user_123")).rejects.toThrow("upstream down");
  });

  it("returns null only when the provider reports the user as not found", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getUser.mockRejectedValue(Object.assign(new Error("gone"), { status: 404 }));

    await expect(provider.getUser("user_123")).resolves.toBeNull();
  });

  it("pages through every member and user of an organization", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getOrganization.mockResolvedValue({
      id: "org_123",
      name: "Org",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    const member = (id: string) => ({
      id: `mem_${id}`,
      userId: id,
      role: { slug: "admin" },
      status: "active",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    const workOsUser = (id: string) => ({
      id,
      email: `${id}@example.com`,
      firstName: null,
      lastName: null,
      profilePictureUrl: null,
    });
    mocks.listOrganizationMemberships
      .mockResolvedValueOnce({ data: [member("user_1")], listMetadata: { after: "m_1" } })
      .mockResolvedValueOnce({ data: [member("user_2")], listMetadata: { after: null } });
    mocks.listUsers
      .mockResolvedValueOnce({ data: [workOsUser("user_1")], listMetadata: { after: "u_1" } })
      .mockResolvedValueOnce({ data: [workOsUser("user_2")], listMetadata: { after: null } });

    const result = await provider.getOrganizationMemberList("org_123");

    expect(result.data.map((item) => item.user.id)).toEqual(["user_1", "user_2"]);
    // A cursor here would imply pages the callers never fetch.
    expect(result.metadata).toEqual({});
  });

  it("keeps a member the user listing missed rather than hiding them", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getOrganization.mockResolvedValue({
      id: "org_123",
      name: "Org",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    mocks.listOrganizationMemberships.mockResolvedValue({
      data: [
        {
          id: "mem_1",
          userId: "user_missed",
          role: { slug: "admin" },
          status: "active",
          createdAt: "2026-01-01",
          updatedAt: "2026-01-01",
        },
      ],
      listMetadata: {},
    });
    mocks.listUsers.mockResolvedValue({ data: [], listMetadata: {} });
    mocks.getUser.mockResolvedValue({
      id: "user_missed",
      email: "missed@example.com",
      firstName: null,
      lastName: null,
      profilePictureUrl: null,
    });

    const result = await provider.getOrganizationMemberList("org_123");

    expect(result.data.map((item) => item.user.id)).toEqual(["user_missed"]);
    expect(mocks.logOperation).not.toHaveBeenCalled();
  });

  it("fails the member list when a backfill hits a provider outage", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getOrganization.mockResolvedValue({
      id: "org_123",
      name: "Org",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    mocks.listOrganizationMemberships.mockResolvedValue({
      data: [
        {
          id: "mem_1",
          userId: "user_unknown",
          role: { slug: "admin" },
          status: "active",
          createdAt: "2026-01-01",
          updatedAt: "2026-01-01",
        },
      ],
      listMetadata: {},
    });
    mocks.listUsers.mockResolvedValue({ data: [], listMetadata: {} });
    mocks.getUser.mockRejectedValue(Object.assign(new Error("upstream down"), { status: 503 }));

    // Silently shortening this list would hide the member from revocation.
    await expect(provider.getOrganizationMemberList("org_123")).rejects.toThrow("upstream down");
  });

  it("drops a member whose user vanished instead of failing the whole list", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getOrganization.mockResolvedValue({
      id: "org_123",
      name: "Org",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    mocks.listOrganizationMemberships.mockResolvedValue({
      data: [
        {
          id: "mem_1",
          userId: "user_gone",
          role: { slug: "admin" },
          status: "active",
          createdAt: "2026-01-01",
          updatedAt: "2026-01-01",
        },
      ],
      listMetadata: {},
    });
    mocks.listUsers.mockResolvedValue({ data: [], listMetadata: {} });
    mocks.getUser.mockRejectedValue(Object.assign(new Error("gone"), { status: 404 }));

    const result = await provider.getOrganizationMemberList("org_123");

    expect(result.data).toEqual([]);
    expect(mocks.logOperation).toHaveBeenCalledWith(
      "warn",
      "WorkOS membership has no matching user",
      expect.objectContaining({ org_id: "org_123", membership_id: "mem_1" }),
    );
  });

  it("stops paging a cursor that never settles", async () => {
    const provider = new WorkOSAuthProvider();
    vi.spyOn(provider, "getUser").mockResolvedValue({
      id: "user_123",
      email: "user@example.com",
      firstName: null,
      lastName: null,
      avatarUrl: null,
      fullName: null,
    });
    mocks.listOrganizationMemberships.mockResolvedValue({
      data: [],
      listMetadata: { after: "never_ends" },
    });

    await expect(provider.listMemberships("user_123")).rejects.toThrow("exceeded 50 pages");
  });

  it("bounds the users leg of the member list too", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getOrganization.mockResolvedValue({
      id: "org_123",
      name: "Org",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    mocks.listOrganizationMemberships.mockResolvedValue({ data: [], listMetadata: {} });
    mocks.listUsers.mockResolvedValue({ data: [], listMetadata: { after: "never_ends" } });

    await expect(provider.getOrganizationMemberList("org_123")).rejects.toThrow(
      "exceeded 50 pages",
    );
  });

  it("pages the two member-list legs independently", async () => {
    const provider = new WorkOSAuthProvider();
    mocks.getOrganization.mockResolvedValue({
      id: "org_123",
      name: "Org",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    const member = (id: string) => ({
      id: `mem_${id}`,
      userId: id,
      role: { slug: "admin" },
      status: "active",
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
    });
    const workOsUser = (id: string) => ({
      id,
      email: `${id}@example.com`,
      firstName: null,
      lastName: null,
      profilePictureUrl: null,
    });
    // One leg is shorter than the other; a shared cursor would truncate.
    mocks.listOrganizationMemberships.mockResolvedValue({
      data: [member("user_1"), member("user_2")],
      listMetadata: {},
    });
    mocks.listUsers
      .mockResolvedValueOnce({ data: [workOsUser("user_1")], listMetadata: { after: "u_1" } })
      .mockResolvedValueOnce({ data: [workOsUser("user_2")], listMetadata: { after: "u_2" } })
      .mockResolvedValueOnce({ data: [], listMetadata: {} });

    const result = await provider.getOrganizationMemberList("org_123");

    expect(result.data.map((item) => item.user.id)).toEqual(["user_1", "user_2"]);
    expect(mocks.listUsers).toHaveBeenCalledTimes(3);
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
