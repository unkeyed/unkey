import type { MembershipListResponse, Organization, UpdateOrgParams, User } from "./types";

/**
 * Provider-neutral administrative operations used by the dashboard.
 *
 * AuthKit owns production authentication, session handling, and MFA. Provider
 * implementations here only cover dashboard-managed users, organizations,
 * memberships, and invitations.
 */
export abstract class BaseAuthProvider {
  abstract getUser(userId: string): Promise<User | null>;
  abstract createTenant(params: { name: string; userId: string }): Promise<string>;
  abstract updateOrg(params: UpdateOrgParams): Promise<Organization>;
  protected abstract createOrg(name: string): Promise<Organization>;
  abstract getOrg(orgId: string): Promise<Organization>;
  abstract listMemberships(userId: string): Promise<MembershipListResponse>;
  abstract getOrganizationMemberList(orgId: string): Promise<MembershipListResponse>;
  abstract deactivateMembership(membershipId: string, orgId: string): Promise<void>;

  protected providerError(error: unknown): Error {
    return error instanceof Error ? error : new Error("Authentication provider request failed");
  }
}
