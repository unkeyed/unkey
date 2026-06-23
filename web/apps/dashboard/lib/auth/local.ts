import { BaseAuthProvider } from "./base-provider";
import { shouldUseSecureCookies } from "./cookie-security";
import { getCookie } from "./cookies";
import {
  type AuthenticatedUser,
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  LOCAL_AUTH_PERMISSIONS,
  LOCAL_ORG_ID,
  LOCAL_ORG_ROLE,
  LOCAL_USER_ID,
  type Membership,
  type MembershipListResponse,
  type MfaEnrollmentStart,
  type MfaFactor,
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
 * - Single organization / single workspace
 * - No-op implementations for multi-user functionality
 */
export class LocalAuthProvider extends BaseAuthProvider {
  private static instance: LocalAuthProvider | null = null;

  // Fixed IDs for local development
  private readonly USER_ID = LOCAL_USER_ID;
  private readonly ORG_ID = LOCAL_ORG_ID;
  private readonly ROLE = LOCAL_ORG_ROLE;

  // Fixed user and org objects
  private readonly user: AuthenticatedUser;
  private readonly organization: Organization;
  private readonly membership: Membership;

  constructor() {
    super();

    const timestamp = new Date().toISOString();

    // Initialize the single user
    this.user = {
      id: this.USER_ID,
      email: "admin@example.com",
      firstName: "Local",
      lastName: "Admin",
      avatarUrl: null,
      fullName: "Local Admin",
      role: this.ROLE,
      orgId: this.ORG_ID,
    };

    // Initialize the single organization
    this.organization = {
      id: this.ORG_ID,
      name: "Local Org",
      createdAt: timestamp,
      updatedAt: timestamp,
    };

    // Initialize the single membership
    this.membership = {
      id: "mem_local",
      user: this.user,
      organization: this.organization,
      role: this.ROLE,
      createdAt: timestamp,
      updatedAt: timestamp,
      status: "active",
    };

    if (!LocalAuthProvider.instance) {
      LocalAuthProvider.instance = this;
    }
  }

  // Session Management
  async validateSession(sessionToken: string): Promise<SessionValidationResult> {
    if (!sessionToken) {
      return { isValid: false, shouldRefresh: false };
    }

    // All local session tokens are valid; no need to refresh
    return {
      isValid: true,
      shouldRefresh: false,
      userId: this.USER_ID,
      orgId: this.ORG_ID,
      permissions: LOCAL_AUTH_PERMISSIONS,
      role: this.ROLE,
      user: this.user,
    };
  }

  async refreshSession(sessionToken: string): Promise<SessionRefreshResult> {
    if (!sessionToken) {
      throw new Error("No session token provided");
    }

    // sessions never need refreshing in local mode
    // Set expiration date very far in the future (effectively never expires)
    const expiresAt = new Date();
    expiresAt.setFullYear(expiresAt.getFullYear() + 100); // 100 years in the future

    return {
      newToken: sessionToken, // Return the same token
      expiresAt,
      session: {
        userId: this.USER_ID,
        orgId: this.ORG_ID,
        permissions: LOCAL_AUTH_PERMISSIONS,
        role: this.ROLE,
        user: this.user,
      },
    };
  }

  // User Management
  async getUser(userId: string): Promise<User | null> {
    if (!userId) {
      throw new Error("User Id is required.");
    }

    // Only return data for the fixed user ID
    if (userId === this.USER_ID) {
      return this.user;
    }
    return null;
  }

  async findUser(email: string): Promise<User | null> {
    if (!email) {
      throw new Error("Email address is required.");
    }

    // Only return data for the fixed user email
    if (email.toLowerCase() === this.user.email.toLowerCase()) {
      return this.user;
    }
    return null;
  }

  // Organization Management
  async createTenant(params: {
    name: string;
    userId: string;
  }): Promise<string> {
    const { name, userId } = params;
    if (!name || !userId) {
      throw new Error("Organization name and userId are required.");
    }

    // In local auth, we just return the existing org ID
    return this.ORG_ID;
  }

  protected async createOrg(name: string): Promise<Organization> {
    if (!name) {
      throw new Error("Organization name is required.");
    }

    // In local auth, we can't create more orgs
    // Just return the existing org
    return this.organization;
  }

  async getOrg(orgId: string): Promise<Organization> {
    if (!orgId) {
      throw new Error("Organization Id is required.");
    }

    // Only return data for the fixed org ID
    if (orgId === this.ORG_ID) {
      return this.organization;
    }
    throw new Error(`Organization ${orgId} not found`);
  }

  async updateOrg(params: UpdateOrgParams): Promise<Organization> {
    // this will reset to the fixed values when the developer restarts the local development server
    const { id, name } = params;
    if (!id || !name) {
      throw new Error("Organization id and name are required.");
    }

    if (id !== this.ORG_ID) {
      throw new Error(`Organization ${id} not found`);
    }

    // Create a new object with updated name and timestamp
    const updatedOrg = {
      ...this.organization,
      name,
      updatedAt: new Date().toISOString(),
    };

    // Update our reference
    this.organization.name = name;
    this.organization.updatedAt = updatedOrg.updatedAt;

    return updatedOrg;
  }

  async switchOrg(newOrgId: string): Promise<SessionRefreshResult> {
    // Get current session token
    const currentToken = await getCookie(UNKEY_SESSION_COOKIE);
    if (!currentToken) {
      throw new Error("No active session found");
    }

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
        role: "admin",
      },
    };
  }

  // Membership Management
  async listMemberships(userId: string): Promise<MembershipListResponse> {
    if (!userId) {
      throw new Error("User Id is required.");
    }

    if (userId === this.USER_ID) {
      return {
        data: [this.membership],
        metadata: {},
      };
    }
    return { data: [], metadata: {} };
  }

  async getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
    if (!orgId) {
      throw new Error("Organization id is required.");
    }

    if (orgId === this.ORG_ID) {
      return {
        data: [this.membership],
        metadata: {},
      };
    }
    return { data: [], metadata: {} };
  }

  async updateMembership(params: UpdateMembershipParams): Promise<Membership> {
    const { membershipId, role, orgId } = params;
    if (!membershipId || !role || !orgId) {
      throw new Error("Membership id, role, and organization id are required.");
    }

    if (membershipId !== this.membership.id) {
      throw new Error(`Membership ${membershipId} not found`);
    }

    // Prevent updating the only membership
    throw new Error("Cannot update the default membership in local development mode");
  }

  async removeMembership(membershipId: string, orgId: string): Promise<void> {
    if (!membershipId || !orgId) {
      throw new Error("Membership id and organization id are required.");
    }

    // Cannot remove the only membership
    if (membershipId === this.membership.id) {
      throw new Error("Cannot remove the default membership");
    }
    throw new Error(`Membership ${membershipId} not found`);
  }

  async deactivateMembership(membershipId: string, orgId: string): Promise<void> {
    if (!membershipId || !orgId) {
      throw new Error("Membership id and organization id are required.");
    }

    if (membershipId === this.membership.id) {
      throw new Error("Cannot deactivate the default membership");
    }
    throw new Error(`Membership ${membershipId} not found`);
  }

  // Invitation Management - No-op methods
  async inviteMember(params: OrgInviteParams): Promise<Invitation> {
    const { orgId, email } = params;
    if (!orgId || !email) {
      throw new Error("Organization id and email are required.");
    }

    // return a mock invitation
    const now = new Date();
    const expiresAt = new Date(now);
    expiresAt.setDate(expiresAt.getDate() + 7);

    return {
      id: `inv_local_${Date.now()}`,
      email,
      state: "pending",
      acceptedAt: null,
      revokedAt: null,
      expiresAt: expiresAt.toISOString(),
      token: `inv_token_${Date.now()}`,
      organizationId: orgId,
      inviterUserId: this.USER_ID,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
  }

  async getInvitationList(orgId: string): Promise<InvitationListResponse> {
    if (!orgId) {
      throw new Error("Organization Id is required");
    }

    // return empty list
    return { data: [], metadata: {} };
  }

  async getInvitation(_invitationToken: string): Promise<Invitation | null> {
    // return null
    return null;
  }

  async revokeOrgInvitation(invitationId: string, orgId: string): Promise<void> {
    if (!invitationId || !orgId) {
      throw new Error("Invitation id and organization id are required.");
    }

    // No-op implementation
    return;
  }

  async acceptInvitation(invitationId: string): Promise<Invitation> {
    if (!invitationId) {
      throw new Error("Invitation Id is required");
    }

    // return a mock accepted invitation
    const now = new Date();

    return {
      id: invitationId,
      email: "invited@example.com",
      state: "accepted",
      acceptedAt: now.toISOString(),
      revokedAt: null,
      expiresAt: now.toISOString(),
      token: "accepted_token",
      organizationId: this.ORG_ID,
      inviterUserId: this.USER_ID,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
  }

  // Authentication Management
  async signUpViaEmail(
    _params: UserData & {
      ipAddress?: string;
      userAgent?: string;
    },
  ): Promise<EmailAuthResult> {
    // always successful
    return { success: true };
  }

  async signInViaEmail(_params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<EmailAuthResult> {
    // always successful
    return { success: true };
  }

  async resendAuthCode(_params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
    radarAuthAttemptId?: string;
  }): Promise<EmailAuthResult> {
    // always successful
    return { success: true };
  }

  async verifyAuthCode(_params: {
    email: string;
    code: string;
    invitationToken?: string;
    ipAddress?: string;
    userAgent?: string;
    radarAuthAttemptId?: string;
  }): Promise<VerificationResult> {
    // always successful
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: shouldUseSecureCookies(),
            httpOnly: true,
            sameSite: "lax",
            path: "/",
            maxAge: 7 * 24 * 60 * 60, // 7 days
          },
        },
      ],
    };
  }

  async verifyEmail(_params: {
    code: string;
    token: string;
  }): Promise<VerificationResult> {
    // always successful
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: shouldUseSecureCookies(),
            httpOnly: true,
            sameSite: "lax",
            path: "/",
            maxAge: 7 * 24 * 60 * 60, // 7 days
          },
        },
      ],
    };
  }

  async completeOrgSelection(_params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult> {
    // always successful
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: shouldUseSecureCookies(),
            httpOnly: true,
            sameSite: "lax",
            path: "/",
            maxAge: 7 * 24 * 60 * 60, // 7 days
          },
        },
      ],
    };
  }

  // MFA and Radar challenges never occur in local mode
  async completeMfaChallenge(_params: {
    code: string;
    challengeId: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    throw new Error("MFA is not supported in local development mode");
  }

  async beginMfaEnrollment(_params: {
    userId: string;
    email: string;
  }): Promise<MfaEnrollmentStart> {
    throw new Error("MFA is not supported in local development mode");
  }

  async verifyMfaEnrollment(_params: {
    challengeId: string;
    code: string;
  }): Promise<boolean> {
    throw new Error("MFA is not supported in local development mode");
  }

  async listMfaFactors(_userId: string): Promise<MfaFactor[]> {
    return [];
  }

  async removeMfaFactor(_factorId: string): Promise<void> {
    throw new Error("MFA is not supported in local development mode");
  }

  async completeRadarEmailChallenge(_params: {
    code: string;
    radarChallengeId: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    throw new Error("Radar challenges are not supported in local development mode");
  }

  async sendRadarSmsCode(_params: {
    userId: string;
    phoneNumber: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<{ verificationId: string; phoneNumber: string }> {
    throw new Error("Radar challenges are not supported in local development mode");
  }

  async completeRadarSmsChallenge(_params: {
    code: string;
    verificationId: string;
    phoneNumber: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    throw new Error("Radar challenges are not supported in local development mode");
  }

  async getSignOutUrl(): Promise<string | null> {
    return "/auth/sign-in";
  }

  async revokeSession(_sessionToken: string): Promise<void> {
    // No-op: local provider has no server-side session to revoke.
  }

  // OAuth Methods
  signInViaOAuth(options: SignInViaOAuthOptions): string {
    return options.redirectUrlComplete;
  }

  async completeOAuthSignIn(_callbackRequest: Request): Promise<OAuthResult> {
    // No-op implementation - always successful
    return {
      success: true,
      redirectTo: "/apis",
      cookies: [
        {
          name: UNKEY_SESSION_COOKIE,
          value: `local_session_${Date.now()}`,
          options: {
            secure: shouldUseSecureCookies(),
            httpOnly: true,
            sameSite: "lax",
          },
        },
      ],
    };
  }
}
