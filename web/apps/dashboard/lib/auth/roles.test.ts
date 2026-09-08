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
    expect(isOrganizationRole("viewer")).toBe(true);
    expect(isOrganizationRole("basic_member")).toBe(false);
  });

  it("keeps legacy members active during migration", () => {
    expect(canAccessWorkspace("basic_member")).toBe(true);
    expect(canMutateWorkspace("basic_member")).toBe(true);
  });

  it("gives viewers read-only workspace access", () => {
    expect(canAccessWorkspace("viewer")).toBe(true);
    expect(canMutateWorkspace("viewer")).toBe(false);
  });

  it("rejects unknown roles", () => {
    expect(canAccessWorkspace("unknown")).toBe(false);
    expect(canMutateWorkspace("unknown")).toBe(false);
    expect(organizationRoleLabel("unknown")).toBe("Unknown role");
  });
});
