import { describe, expect, it } from "vitest";
import { assertWorkspaceAdmin } from "./authorize";
import { AgentSignupError } from "./errors";

const orgId = "org_1";

describe("assertWorkspaceAdmin", () => {
  it("allows an active admin of the workspace organization", () => {
    expect(() =>
      assertWorkspaceAdmin([{ organizationId: orgId, role: "admin", status: "active" }], orgId),
    ).not.toThrow();
  });

  it("rejects a non-member with 403", () => {
    expect(() => assertWorkspaceAdmin([], orgId)).toThrow(
      expect.objectContaining({ status: 403, code: "forbidden" }),
    );
    expect(() =>
      assertWorkspaceAdmin(
        [{ organizationId: "org_other", role: "admin", status: "active" }],
        orgId,
      ),
    ).toThrow(expect.objectContaining({ status: 403, code: "forbidden" }));
  });

  it("rejects a non-admin member with 403", () => {
    try {
      assertWorkspaceAdmin([{ organizationId: orgId, role: "developer", status: "active" }], orgId);
    } catch (error) {
      expect(error).toMatchObject({ status: 403, code: "forbidden" });
      return;
    }
    throw new Error("expected a 403");
  });

  it("rejects an inactive admin", () => {
    expect(() =>
      assertWorkspaceAdmin([{ organizationId: orgId, role: "admin", status: "inactive" }], orgId),
    ).toThrow(AgentSignupError);
  });
});
