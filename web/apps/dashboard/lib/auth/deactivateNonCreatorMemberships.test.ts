import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getOrganizationMemberList: vi.fn(),
  deactivateMembership: vi.fn(),
}));

vi.mock("./server", () => ({ auth: mocks }));

import { deactivateNonCreatorMemberships } from "./deactivateNonCreatorMemberships";

describe("deactivateNonCreatorMemberships", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getOrganizationMemberList.mockResolvedValue({ data: [], metadata: {} });
  });

  it("preserves the earliest member and deactivates teammates", async () => {
    mocks.getOrganizationMemberList.mockResolvedValue({
      data: [
        { id: "member_2", createdAt: "2026-02-01T00:00:00.000Z" },
        { id: "creator", createdAt: "2026-01-01T00:00:00.000Z" },
      ],
      metadata: {},
    });
    await deactivateNonCreatorMemberships("org_1");

    expect(mocks.deactivateMembership).toHaveBeenCalledOnce();
    expect(mocks.deactivateMembership).toHaveBeenCalledWith("member_2", "org_1");
  });

  it("does not deactivate members when membership listing fails", async () => {
    mocks.getOrganizationMemberList.mockRejectedValue(new Error("unavailable"));
    const error = vi.spyOn(console, "error").mockImplementation(() => {});

    await deactivateNonCreatorMemberships("org_1");

    expect(mocks.deactivateMembership).not.toHaveBeenCalled();
    error.mockRestore();
  });
});
