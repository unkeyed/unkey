import { BaseAuthProvider } from "./base-provider";
import {
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  type Membership,
  type MembershipListResponse,
  type OAuthResult,
  type OrgInviteParams,
  type Organization,
  type SessionRefreshResult,
  type SessionValidationResult,
  type SignInViaOAuthOptions,
  UNKEY_SESSION_COOKIE,
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
  type UserData,
  type VerificationResult,
} from "./types";

/**
 * Local Auth Provider
 * - Single user (always signed in)
 * - Single organization
 * - No invitations or multi-user support
 */
export class LocalAuthProvider extends BaseAuthProvider {
  private static instance: LocalAuthProvider | null = null;
  
  // Fixed IDs for local development
  private readonly USER_ID = "user_local_admin";
  private readonly ORG_ID = "org_local_default";
  
  // Fixed user and org objects
  private readonly user: User;
  private readonly organization: Organization;
  private readonly membership: Membership;

  constructor() {
    super();

    const timestamp = new Date().toISOString();
    
    // Initialize the single user
    this.user = {
      id: this.USER_ID,
      orgId: this.ORG_ID,
      email: "admin@example.com",
      firstName: "Local",
      lastName: "Admin",
      avatarUrl: null,
      fullName: "Local Admin"
    };

    // Initialize the single organization
    this.organization = {
      id: this.ORG_ID,
      name: "Local Development Org",
      createdAt: timestamp,
      updatedAt: timestamp
    };

    // Initialize the single membership
    this.membership = {
      id: "mem_local",
      user: this.user,
      organization: this.organization,
      role: "admin",
      createdAt: timestamp,
      updatedAt: timestamp,
      status: "active"
    };

    LocalAuthProvider.instance = this;
  }

  // Session Management - Always return a valid session
  async validateSession(_sessionToken: string): Promise<SessionValidationResult> {
    return {
      isValid: true,
      shouldRefresh: false,
      userId: this.USER_ID,
      orgId: this.ORG_ID
    };
  }

  async refreshSession(_sessionToken: string): Promise<SessionRefreshResult> {
    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + 7);

    return {
      newToken: `local_session_${Date.now()}`,
      expiresAt,
      session: {
        userId: this.USER_ID,
        orgId: this.ORG_ID,
      },
    };
  }

  // User Management - Always return the single user
  async getCurrentUser(): Promise<User | null> {
    return this.user;
  }

  async getUser(userId: string): Promise<User | null> {
    // Only return data for the fixed user ID
    if (userId === this.USER_ID) {
      return this.user;
    }
    return null;
  }

  async findUser(email: string): Promise<User | null> {
    // Only return data for the fixed user email
    if (email.toLowerCase() === this.user.email.toLowerCase()) {
      return this.user;
    }
    return null;
  }

  // Organization Management - Always return the single org
  async createTenant(params: { name: string; userId: string }): Promise<string> {
    // In local auth, we can't create more orgs
    // Just return the existing org ID
    return this.ORG_ID;
  }

  protected async createOrg(name: string): Promise<Organization> {
    // In local auth, we can't create more orgs
    // Just return the existing org
    return this.organization;
  }

  async getOrg(orgId: string): Promise<Organization> {
    // Only return data for the fixed org ID
    if (orgId === this.ORG_ID) {
      return this.organization;
    }
    throw new Error(`Organization ${orgId} not found`);
  }

  // this will not persist upon stopping and restarting
  async updateOrg(params: UpdateOrgParams): Promise<Organization> {
    // Allow updating the name of the single org
    const { id, name } = params;
    
    if (id !== this.ORG_ID) {
      throw new Error(`Organization ${id} not found`);
    }
    
    // Create a new object with updated name and timestamp
    const updatedOrg = {
      ...this.organization,
      name,
      updatedAt: new Date().toISOString()
    };
    
    // Update our reference
    (this.organization as any).name = name;
    (this.organization as any).updatedAt = updatedOrg.updatedAt;
    
    return updatedOrg;
  }

  async switchOrg(newOrgId: string): Promise<SessionRefreshResult> {
    // In local auth we can't switch orgs
    // If the requested org is our fixed org, return success
    if (newOrgId !== this.ORG_ID) {
      throw new Error(`Organization ${newOrgId} not found`);
    }
    
    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + 7);

    return {
      newToken: `local_session_${Date.now()}`,
      expiresAt,
      session: {
        userId: this.USER_ID,
        orgId: this.ORG_ID,
      },
    };
  }

  // Membership Management - Only return the single membership
  async listMemberships(userId: string): Promise<MembershipListResponse> {
    if (userId === this.USER_ID) {
      return {
        data: [this.membership],
        metadata: {},
      };
    }
    return { data: [], metadata: {} };
  }

  async getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
    if (orgId === this.ORG_ID) {
      return {
        data: [this.membership],
        metadata: {},
      };
    }
    return { data: [], metadata: {} };
  }

  async updateMembership(params: UpdateMembershipParams): Promise<Membership> {
    const { membershipId, role } = params;
    
    if (membershipId !== this.membership.id) {
      throw new Error(`Membership ${membershipId} not found`);
    }
    
    // Create a new membership with updated role
    const updatedMembership = {
      ...this.membership,
      role,
      updatedAt: new Date().toISOString()
    };
    
    // Update our reference
    (this.membership as any).role = role;
    (this.membership as any).updatedAt = updatedMembership.updatedAt;
    
    return updatedMembership;
  }

  async removeMembership(membershipId: string): Promise<void> {
    // Cannot remove the only membership
    if (membershipId === this.membership.id) {
      throw new Error("Cannot remove the default membership");
    }
    throw new Error(`Membership ${membershipId} not found`);
  }

  // Invitation Management - No-op implementations returning empty data
  async inviteMember(params: OrgInviteParams): Promise<Invitation> {
    throw new Error("Invitations are not supported in local development mode");
  }

  async getInvitationList(orgId: string): Promise<InvitationListResponse> {
    return { data: [], metadata: {} };
  }

  async getInvitation(invitationToken: string): Promise<Invitation | null> {
    return null;
  }

  async revokeOrgInvitation(invitationId: string): Promise<void> {
    throw new Error("Invitations are not supported in local development mode");
  }

  async acceptInvitation(invitationId: string): Promise<Invitation> {
    throw new Error("Invitations are not supported in local development mode");
  }

  // Authentication Management - All simplistic implementations
  async signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
    return { success: true };
  }

  async signInViaEmail(email: string): Promise<EmailAuthResult> {
    return { success: true };
  }

  async resendAuthCode(email: string): Promise<EmailAuthResult> {
    return { success: true };
  }

  async verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
  }): Promise<VerificationResult> {
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: true,
            httpOnly: true,
            sameSite: "lax",
          },
        },
      ],
    };
  }

  async verifyEmail(params: {
    code: string;
    token: string;
  }): Promise<VerificationResult> {
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: true,
            httpOnly: true,
            sameSite: "lax",
          },
        },
      ],
    };
  }

  async completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult> {
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: true,
            httpOnly: true,
            sameSite: "lax",
          },
        },
      ],
    };
  }

  async getSignOutUrl(): Promise<string | null> {
    return null;
  }

  // OAuth Methods - Simplified
  signInViaOAuth(options: SignInViaOAuthOptions): string {
    return options.redirectUrlComplete;
  }

  async completeOAuthSignIn(_callbackRequest: Request): Promise<OAuthResult> {
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: true,
            httpOnly: true,
            sameSite: "lax",
          },
        },
      ],
    };
  }
}