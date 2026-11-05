import { describe, expect, it } from "vitest";
import {
  canAccessWorkspace,
  canMutateWorkspace,
  isOrganizationRole,
  organizationRoleLabel,
} from "./roles";

describe("organization roles", () => {
  it("recognizes roles that the dashboard can assign", () => {
    expect(isOrganizationRole("admin")).toBe(true);
    expect(isOrganizationRole("developer")).toBe(true);
    expect(isOrganizationRole("viewer")).toBe(false);
    expect(isOrganizationRole("basic_member")).toBe(false);
    expect(isOrganizationRole("member")).toBe(false);
  });

  it("treats both legacy member roles as developers", () => {
    for (const role of ["basic_member", "member"]) {
      expect(canAccessWorkspace(role)).toBe(true);
      expect(canMutateWorkspace(role)).toBe(true);
      expect(organizationRoleLabel(role)).toBe("Developer");
    }
  });

  it("rejects unknown roles", () => {
    expect(canAccessWorkspace("viewer")).toBe(false);
    expect(canMutateWorkspace("viewer")).toBe(false);
    expect(organizationRoleLabel("viewer")).toBe("Unknown role");
    expect(canAccessWorkspace("unknown")).toBe(false);
    expect(canMutateWorkspace("unknown")).toBe(false);
    expect(organizationRoleLabel("unknown")).toBe("Unknown role");
  });
});
