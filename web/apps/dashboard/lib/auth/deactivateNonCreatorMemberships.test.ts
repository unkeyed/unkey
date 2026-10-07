import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getOrganizationMemberList: vi.fn(),
  deactivateMembership: vi.fn(),
  logOperation: vi.fn(),
}));

vi.mock("./server", () => ({ auth: mocks }));
vi.mock("@/lib/logging", () => ({ logOperation: mocks.logOperation }));

const membersFrom = (count: number) =>
  Array.from({ length: count }, (_, index) => ({
    id: `member_${index}`,
    // Ascending, so member_0 is the creator the function must preserve.
    createdAt: `2026-01-${String(index + 1).padStart(2, "0")}T00:00:00.000Z`,
  }));

import { deactivateNonCreatorMemberships } from "./deactivateNonCreatorMemberships";

describe("deactivateNonCreatorMemberships", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.getOrganizationMemberList.mockResolvedValue({ data: [], metadata: {} });
    mocks.deactivateMembership.mockResolvedValue(undefined);
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
    expect(mocks.logOperation).toHaveBeenCalledWith(
      "error",
      "Membership deactivation could not list members",
      expect.objectContaining({ org_id: "org_1" }),
    );
    error.mockRestore();
  });

  it("deactivates every member past the batch boundary", async () => {
    // More than CONCURRENCY, so the batching loop runs more than one round.
    mocks.getOrganizationMemberList.mockResolvedValue({
      data: membersFrom(47),
      metadata: {},
    });

    await deactivateNonCreatorMemberships("org_1");

    expect(mocks.deactivateMembership).toHaveBeenCalledTimes(46);
    const deactivated = mocks.deactivateMembership.mock.calls.map(([id]) => id);
    expect(deactivated).not.toContain("member_0");
    expect(new Set(deactivated).size).toBe(46);
  });

  it("reports the org and the running counts when a member fails to deactivate", async () => {
    mocks.getOrganizationMemberList.mockResolvedValue({
      data: membersFrom(4),
      metadata: {},
    });
    mocks.deactivateMembership.mockImplementation(async (membershipId: string) => {
      if (membershipId === "member_2") {
        throw new Error("rate limited");
      }
    });
    const error = vi.spyOn(console, "error").mockImplementation(() => {});

    await deactivateNonCreatorMemberships("org_1");

    expect(mocks.logOperation).toHaveBeenCalledWith(
      "error",
      "Membership deactivation left members active",
      expect.objectContaining({ org_id: "org_1", failed_count: 1, total_count: 3 }),
    );
    error.mockRestore();
  });

  it("stays quiet when every deactivation succeeds", async () => {
    mocks.getOrganizationMemberList.mockResolvedValue({
      data: membersFrom(4),
      metadata: {},
    });

    await deactivateNonCreatorMemberships("org_1");

    expect(mocks.logOperation).not.toHaveBeenCalled();
  });
});
