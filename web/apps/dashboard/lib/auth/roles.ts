export const ORGANIZATION_ROLES = ["admin", "developer"] as const;

export type OrganizationRole = (typeof ORGANIZATION_ROLES)[number];

const LEGACY_MEMBER_ROLES: readonly string[] = ["basic_member", "member"];

export function isLegacyMemberRole(role: string | null | undefined): boolean {
  return LEGACY_MEMBER_ROLES.some((legacyRole) => legacyRole === role);
}

export function isOrganizationRole(role: string | null | undefined): role is OrganizationRole {
  return ORGANIZATION_ROLES.some((organizationRole) => organizationRole === role);
}

export function canAccessWorkspace(role: string | null | undefined): boolean {
  return isOrganizationRole(role) || isLegacyMemberRole(role);
}

export function canMutateWorkspace(role: string | null | undefined): boolean {
  return canAccessWorkspace(role);
}

export function organizationRoleLabel(role: string | null | undefined): string {
  if (role === "admin") {
    return "Admin";
  }
  if (role === "developer" || isLegacyMemberRole(role)) {
    return "Developer";
  }
  return "Unknown role";
}
