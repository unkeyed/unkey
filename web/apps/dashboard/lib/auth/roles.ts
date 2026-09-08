export const ORGANIZATION_ROLES = ["admin", "developer", "viewer"] as const;

export type OrganizationRole = (typeof ORGANIZATION_ROLES)[number];

const LEGACY_MEMBER_ROLE = "basic_member";

export function isOrganizationRole(role: string | null | undefined): role is OrganizationRole {
  return ORGANIZATION_ROLES.some((organizationRole) => organizationRole === role);
}

export function canAccessWorkspace(role: string | null | undefined): boolean {
  return isOrganizationRole(role) || role === LEGACY_MEMBER_ROLE;
}

export function canMutateWorkspace(role: string | null | undefined): boolean {
  return canAccessWorkspace(role) && role !== "viewer";
}

export function organizationRoleLabel(role: string | null | undefined): string {
  switch (role) {
    case "admin":
      return "Admin";
    case "developer":
    case LEGACY_MEMBER_ROLE:
      return "Developer";
    case "viewer":
      return "Viewer";
    default:
      return "Unknown role";
  }
}
