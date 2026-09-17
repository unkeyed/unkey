import { BaseAuthProvider } from "./base-provider";
import { getCookie } from "./cookies";
import {
  type AuthenticatedUser,
  type Invitation,
  type InvitationListResponse,
  LOCAL_AUTH_PERMISSIONS,
  LOCAL_ORG_ID,
  LOCAL_ORG_ROLE,
  LOCAL_USER_ID,
  type Membership,
  type MembershipListResponse,
  type OrgInviteParams,
  type Organization,
  type SessionRefreshResult,
  type SessionValidationResult,
  UNKEY_SESSION_COOKIE,
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
} from "./types";

/**
 * Local mode uses one fixed account and organization. It intentionally does
 * not emulate hosted authentication, MFA, or invitation acceptance.
 */
export class LocalAuthProvider extends BaseAuthProvider {
  private readonly user: AuthenticatedUser;
  private readonly organization: Organization;
  private readonly membership: Membership;

  constructor() {
    super();
    const timestamp = new Date().toISOString();
    this.user = {
      id: LOCAL_USER_ID,
      email: "admin@example.com",
      firstName: "Local",
      lastName: "Admin",
      avatarUrl: null,
      fullName: "Local Admin",
      role: LOCAL_ORG_ROLE,
      orgId: LOCAL_ORG_ID,
    };
    this.organization = {
      id: LOCAL_ORG_ID,
      name: "Local Org",
      createdAt: timestamp,
      updatedAt: timestamp,
    };
    this.membership = {
      id: "mem_local",
      user: this.user,
      organization: this.organization,
      role: LOCAL_ORG_ROLE,
      createdAt: timestamp,
      updatedAt: timestamp,
      status: "active",
    };
  }

  async validateSession(sessionToken: string): Promise<SessionValidationResult> {
    if (!sessionToken) {
      return { isValid: false };
    }
    return {
      isValid: true,
      userId: LOCAL_USER_ID,
      orgId: LOCAL_ORG_ID,
      permissions: LOCAL_AUTH_PERMISSIONS,
      role: LOCAL_ORG_ROLE,
      user: this.user,
    };
  }

  async switchOrg(newOrgId: string): Promise<SessionRefreshResult> {
    const currentToken = await getCookie(UNKEY_SESSION_COOKIE);
    if (!currentToken) {
      throw new Error("No active session found");
    }
    if (newOrgId !== LOCAL_ORG_ID) {
      throw new Error(`Organization ${newOrgId} not found`);
    }
    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + 7);
    return {
      newToken: currentToken,
      expiresAt,
    };
  }

  async getUser(userId: string): Promise<User | null> {
    return userId === LOCAL_USER_ID ? this.user : null;
  }

  async createTenant(params: { name: string; userId: string }): Promise<string> {
    if (!params.name || !params.userId) {
      throw new Error("Organization name and userId are required.");
    }
    return LOCAL_ORG_ID;
  }

  protected async createOrg(name: string): Promise<Organization> {
    if (!name) {
      throw new Error("Organization name is required.");
    }
    return this.organization;
  }

  async getOrg(orgId: string): Promise<Organization> {
    if (orgId !== LOCAL_ORG_ID) {
      throw new Error(`Organization ${orgId} not found`);
    }
    return this.organization;
  }

  async updateOrg(params: UpdateOrgParams): Promise<Organization> {
    if (params.id !== LOCAL_ORG_ID || !params.name) {
      throw new Error(`Organization ${params.id} not found`);
    }
    this.organization.name = params.name;
    this.organization.updatedAt = new Date().toISOString();
    return { ...this.organization };
  }

  async listMemberships(userId: string): Promise<MembershipListResponse> {
    return { data: userId === LOCAL_USER_ID ? [this.membership] : [], metadata: {} };
  }

  async getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
    return { data: orgId === LOCAL_ORG_ID ? [this.membership] : [], metadata: {} };
  }

  async updateMembership(params: UpdateMembershipParams): Promise<Membership> {
    if (params.membershipId !== this.membership.id) {
      throw new Error(`Membership ${params.membershipId} not found`);
    }
    throw new Error("Cannot update the default membership in local development mode");
  }

  async removeMembership(membershipId: string, _orgId: string): Promise<void> {
    if (membershipId === this.membership.id) {
      throw new Error("Cannot remove the default membership");
    }
    throw new Error(`Membership ${membershipId} not found`);
  }

  async deactivateMembership(membershipId: string, _orgId: string): Promise<void> {
    if (membershipId === this.membership.id) {
      throw new Error("Cannot deactivate the default membership");
    }
    throw new Error(`Membership ${membershipId} not found`);
  }

  async inviteMember(params: OrgInviteParams): Promise<Invitation> {
    const now = new Date();
    const expiresAt = new Date(now);
    expiresAt.setDate(expiresAt.getDate() + 7);
    return {
      id: `inv_local_${now.getTime()}`,
      email: params.email,
      state: "pending",
      expiresAt: expiresAt.toISOString(),
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
  }

  async getInvitationList(_orgId: string): Promise<InvitationListResponse> {
    return { data: [], metadata: {} };
  }

  async revokeOrgInvitation(_invitationId: string, _orgId: string): Promise<void> {}
}

export const localAuth = new LocalAuthProvider();
