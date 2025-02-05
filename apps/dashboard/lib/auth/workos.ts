import { WorkOS, User as WorkOSUser, Organization as WorkOSOrganization, Invitation as WorkOSInvitation } from "@workos-inc/node";
import { env } from "@/lib/env";
import { BaseAuthProvider } from "./base-provider";
import { getCookie, updateCookie } from "./cookies";
import {
  SessionValidationResult,
  SessionData,
  UNKEY_SESSION_COOKIE,
  AuthErrorCode,
  User,
  Organization,
  UpdateOrgParams,
  MembershipListResponse,
  UpdateMembershipParams,
  Membership,
  OrgInviteParams,
  Invitation,
  InvitationListResponse,
  UserData,
  EmailAuthResult,
  VerificationResult,
  SignInViaOAuthOptions,
  OAuthResult,
  PENDING_SESSION_COOKIE,
} from "./types";

export class WorkOSAuthProvider extends BaseAuthProvider {
  private static instance: WorkOSAuthProvider | null = null;
  private readonly provider: WorkOS;
  private readonly clientId: string;
  private readonly cookiePassword: string;

  constructor(config: { apiKey: string; clientId: string }) {
    super();
    
    const cookiePassword = env().WORKOS_COOKIE_PASSWORD;
    if (!cookiePassword) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required for WorkOS authentication");
    }

    // Initialize properties after validation
    this.clientId = config.clientId;
    this.cookiePassword = cookiePassword;  // TypeScript now knows this is string
    this.provider = new WorkOS(config.apiKey, { clientId: config.clientId });

    WorkOSAuthProvider.instance = this;
  }

  // Session Management
  async validateSession(sessionToken: string): Promise<SessionValidationResult> {
    if (!sessionToken) {
      return { isValid: false, shouldRefresh: false };
    }

    try {
      const session = await this.provider.userManagement.loadSealedSession({
        sessionData: sessionToken,
        cookiePassword: this.cookiePassword
      });

      const authResult = await session.authenticate();

      if (authResult.authenticated) {
        return {
          isValid: true,
          shouldRefresh: false,
          userId: authResult.user.id,
          orgId: authResult.organizationId ?? null
        };
      }

      return { isValid: false, shouldRefresh: true };
    } catch (error) {
      console.error('Session validation error:', {
        error: error instanceof Error ? error.message : 'Unknown error',
        token: sessionToken.substring(0, 10) + '...'
      });
      return { isValid: false, shouldRefresh: false };
    }
  }

  async refreshSession(orgId?: string): Promise<SessionData | null> {
    const token = await getCookie(UNKEY_SESSION_COOKIE);
    if (!token) {
      console.error("No session found");
      return null;
    }

    try {
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: this.cookiePassword
      });

      const refreshResult = await session.refresh({
        cookiePassword: this.cookiePassword,
        ...(orgId && { organizationId: orgId })
      });

      if (refreshResult.authenticated && refreshResult.session) {
        await updateCookie(UNKEY_SESSION_COOKIE, refreshResult.sealedSession);
        return {
          userId: refreshResult.session.user.id,
          orgId: refreshResult.session.organizationId ?? null
        };
      }

      await updateCookie(
        UNKEY_SESSION_COOKIE, 
        null, 
        'reason' in refreshResult ? refreshResult.reason : 'Session refresh failed');
      return null;
    } catch (error) {
      console.error('Session refresh error:', {
        error: error instanceof Error ? error.message : 'Unknown error',
        token: token.substring(0, 10) + '...'
      });
      return null;
    }
  }

  // User Management
  async getCurrentUser(): Promise<User | null> {
    try {
      const token = await getCookie(UNKEY_SESSION_COOKIE);
      if (!token) return null;
  
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: this.cookiePassword
      });
  
      const authResult = await session.authenticate();
      if (!authResult.authenticated) {
        console.error("Get current user failed:", authResult.reason);
        return null;
      }
  
      const { user, organizationId } = authResult;
      return this.transformUserData({
        ...user,
        organizationId: organizationId
      });
  
    } catch (error) {
      console.error('Error getting current user:', error);
      return null;
    }
  }
  
  async getUser(userId: string): Promise<User | null> {
    if (!userId) {
      throw new Error("User Id is required.");
    }
  
    try {
      const user = await this.provider.userManagement.getUser(userId);
      if (!user) {
        return null;
      }
  
      return this.transformUserData(user);
    } catch (error) {
      console.error("Failed to get user:", error);
      return null;
    }
  }

  // Organization Management
  async createTenant(params: { name: string; userId: string }): Promise<string> {
    const { name, userId } = params;
    if (!name || !userId) {
      throw new Error('Organization name and userId are required.');
    }

    try {
      const org = await this.createOrg(name);
      const membership = await this.provider.userManagement.createOrganizationMembership({
        organizationId: org.id,
        userId,
        roleSlug: "admin"
      });

      return membership.organizationId;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  protected async createOrg(name: string): Promise<Organization> {
    if (!name) {
      throw new Error("Organization name is required.");
    }

    try {
      const org = await this.provider.organizations.createOrganization({ name });
      return this.transformOrganizationData(org);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getOrg(orgId: string): Promise<Organization> {
    if (!orgId) {
      throw new Error("Organization Id is required.");
    }

    try {
      const org = await this.provider.organizations.getOrganization(orgId);
      return this.transformOrganizationData(org);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async updateOrg(params: UpdateOrgParams): Promise<Organization> {
    const { id, name } = params;
    if (!id || !name) {
      throw new Error("Organization id and name are required.");
    }

    try {
      const org = await this.provider.organizations.updateOrganization({
        organization: id,
        name
      });
      return this.transformOrganizationData(org);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // Membership Management
  async listMemberships(): Promise<MembershipListResponse> {
    try {
      const user = await this.getCurrentUser();
      if (!user) {
        return { data: [], metadata: {} };
      }

      const memberships = await this.provider.userManagement.listOrganizationMemberships({
        userId: user.id,
        limit: 100,
        statuses: ["active"]
      });

      // Fetch organizations for each membership
      const orgs = await Promise.all(
        memberships.data.map(m => this.getOrg(m.organizationId))
      );

      const orgMap = new Map(orgs.map(org => [org.id, org]));

      return {
        data: memberships.data.map(membership => ({
          id: membership.id,
          user,
          organization: orgMap.get(membership.organizationId)!,
          role: membership.role.slug,
          createdAt: membership.createdAt,
          updatedAt: membership.updatedAt,
          status: membership.status
        })),
        metadata: memberships.listMetadata || {}
      };
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
    if (!orgId) {
      throw new Error("Organization id is required.");
    }

    try {
      const org = await this.getOrg(orgId);
      const members = await this.provider.userManagement.listOrganizationMemberships({
        organizationId: orgId,
        limit: 100,
        statuses: ["active"]
      });

      // Get user data for each member
      const users = await Promise.all(
        members.data.map(m => this.getUser(m.userId))
      );

      // Create user map excluding null results
      const userMap = new Map(
        users.filter((user): user is User => user !== null)
          .map(user => [user.id, user])
      );

      return {
        data: members.data.map(member => {
          const user = userMap.get(member.userId);
          if (!user) {
            throw new Error(`User ${member.userId} not found`);
          }

          return {
            id: member.id,
            user,
            organization: org,
            role: member.role.slug,
            createdAt: member.createdAt,
            updatedAt: member.updatedAt,
            status: member.status
          };
        }),
        metadata: members.listMetadata || {}
      };
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async updateMembership(params: UpdateMembershipParams): Promise<Membership> {
    const { membershipId, role } = params;
    if (!membershipId || !role) {
      throw new Error("Membership id and role are required.");
    }

    try {
      const membership = await this.provider.userManagement.updateOrganizationMembership(
        membershipId,
        { roleSlug: role }
      );

      // Get related data
      const [org, user] = await Promise.all([
        this.getOrg(membership.organizationId),
        this.getUser(membership.userId)
      ]);

      if (!user) {
        throw new Error(`User ${membership.userId} not found`);
      }

      return {
        id: membership.id,
        user,
        organization: org,
        role: membership.role.slug,
        createdAt: membership.createdAt,
        updatedAt: membership.updatedAt,
        status: membership.status
      };
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async removeMembership(membershipId: string): Promise<void> {
    if (!membershipId) {
      throw new Error("Membership Id is required");
    }

    try {
      await this.provider.userManagement.deleteOrganizationMembership(membershipId);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // Invitation Management
  async inviteMember(params: OrgInviteParams): Promise<Invitation> {
    const { orgId, email, role = "basic_member" } = params;
    if (!orgId || !email) {
      throw new Error("Organization id and email are required.");
    }

    try {
      const user = await this.getCurrentUser();
      if (!user) {
        throw new Error("User must be authenticated to invite members.");
      }

      const invitation = await this.provider.userManagement.sendInvitation({
        email,
        organizationId: orgId,
        roleSlug: role,
        inviterUserId: user.id
      });

      return this.transformInvitationData(invitation, { orgId, inviterId: user.id });
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getInvitationList(orgId: string): Promise<InvitationListResponse> {
    if (!orgId) {
      throw new Error("Organization Id is required");
    }

    try {
      const invitationsList = await this.provider.userManagement.listInvitations({
        organizationId: orgId
      });

      return {
        data: invitationsList.data.map(invitation => 
          this.transformInvitationData(invitation, { orgId })
        ),
        metadata: invitationsList.listMetadata || {}
      };
    } catch (error) {
      console.error("Failed to get organization invitations list:", error);
      return {
        data: [],
        metadata: {}
      };
    }
  }

  async revokeOrgInvitation(invitationId: string): Promise<void> {
    if (!invitationId) {
      throw new Error("Invitation Id is required");
    }

    try {
      await this.provider.userManagement.revokeInvitation(invitationId);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // Authentication Management
  
  async signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
    const { email, firstName, lastName } = params;
    
    try {
      // Create the user with WorkOS
      await this.provider.userManagement.createUser({
        firstName,
        lastName,
        email
      });
  
      // Send magic auth email
      await this.provider.userManagement.createMagicAuth({ email });
  
      return { success: true };
    } catch (error: any) {
      if (error.errors?.some((detail: any) => detail.code === 'email_not_available')) {
        return this.handleError(new Error(AuthErrorCode.EMAIL_ALREADY_EXISTS));
      }
      if (error.message.includes("email_required")) {
        return this.handleError(new Error(AuthErrorCode.INVALID_EMAIL));
      }
      if (error.code === 'user_creation_error') {
        return this.handleError(new Error(AuthErrorCode.USER_CREATION_FAILED));
      }
      return this.handleError(error);
    }
  }
  
  async signInViaEmail(email: string): Promise<EmailAuthResult> {
    try {
      const { data } = await this.provider.userManagement.listUsers({ email });
  
      if (data.length === 0) {
        return this.handleError(new Error(AuthErrorCode.ACCOUNT_NOT_FOUND));
      }
  
      await this.provider.userManagement.createMagicAuth({ email });
      return { success: true };
    } catch (error) {
      return this.handleError(error);
    }
  }
  
  async resendAuthCode(email: string): Promise<EmailAuthResult> {
    try {
      await this.provider.userManagement.createMagicAuth({ email });
      return { success: true };
    } catch (error) {
      return this.handleError(error);
    }
  }
  
  async verifyAuthCode(params: { email: string; code: string }): Promise<VerificationResult> {
    const { email, code } = params;
  
    try {
      const { sealedSession } = await this.provider.userManagement.authenticateWithMagicAuth({
        clientId: this.clientId,
        code,
        email,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword
        }
      });
  
      if (!sealedSession) {
        throw new Error('No sealed session returned');
      }
  
      return {
        success: true,
        redirectTo: '/apis',
        cookies: [{
          name: UNKEY_SESSION_COOKIE,
          value: sealedSession,
          options: {
            secure: true,
            httpOnly: true
          }
        }]
      };
    } catch (error: any) {
      // Handle organization selection required case
      if (error.code === 'organization_selection_required') {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: error.message,
          user: this.transformUserData(error.user),
          organizations: error.organizations.map(this.transformOrganizationData),
          cookies: [{
            name: PENDING_SESSION_COOKIE,
            value: error.pending_authentication_token,
            options: {
              secure: true,
              httpOnly: true
            }
          }]
        };
      }
      return this.handleError(error);
    }
  }
  
  async completeOrgSelection(params: { orgId: string; pendingAuthToken: string }): Promise<VerificationResult> {
    try {
      const { sealedSession } = await this.provider.userManagement.authenticateWithOrganizationSelection({
        pendingAuthenticationToken: params.pendingAuthToken,
        organizationId: params.orgId,
        clientId: this.clientId,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword
        }
      });
  
      if (!sealedSession) {
        throw new Error('No sealed session returned');
      }
  
      return {
        success: true,
        redirectTo: '/apis',
        cookies: [{
          name: UNKEY_SESSION_COOKIE,
          value: sealedSession,
          options: {
            secure: true,
            httpOnly: true
          }
        }]
      };
    } catch (error) {
      return this.handleError(error);
    }
  }

  async getSignOutUrl(): Promise<string | null> {
    const token = await getCookie(UNKEY_SESSION_COOKIE);
    if (!token) {
      return null;
    }
  
    try {
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: this.cookiePassword
      });
  
      return await session.getLogoutUrl();
    } catch (error) {
      console.error('Failed to get sign out URL:', {
        error: error instanceof Error ? error.message : 'Unknown error'
      });
      return null;
    }
  }
  
  // OAuth Methods
  signInViaOAuth(options: SignInViaOAuthOptions): string {
    const { redirectUrl, provider, redirectUrlComplete } = options;
    const state = encodeURIComponent(JSON.stringify({ redirectUrlComplete }));
  
    return this.provider.userManagement.getAuthorizationUrl({
      clientId: this.clientId,
      redirectUri: redirectUrl ?? env().NEXT_PUBLIC_WORKOS_REDIRECT_URI,
      provider: provider === "github" ? "GitHubOAuth" : "GoogleOAuth",
      state
    });
  }
  
  async completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult> {
    const url = new URL(callbackRequest.url);
    const code = url.searchParams.get('code');
    const state = url.searchParams.get('state');
  
    if (!code) {
      return this.handleError(new Error(AuthErrorCode.MISSING_REQUIRED_FIELDS));
    }
  
    try {
      const { sealedSession } = await this.provider.userManagement.authenticateWithCode({
        clientId: this.clientId,
        code,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword
        }
      });
  
      if (!sealedSession) {
        throw new Error('No sealed session returned');
      }
  
      const redirectUrlComplete = state 
        ? JSON.parse(decodeURIComponent(state)).redirectUrlComplete 
        : '/apis';
  
      return {
        success: true,
        redirectTo: redirectUrlComplete,
        cookies: [{
          name: UNKEY_SESSION_COOKIE,
          value: sealedSession,
          options: {
            secure: true,
            httpOnly: true
          }
        }]
      };
    } catch (error: any) {
      if (error.rawData.code === 'organization_selection_required') {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: error.message,
          user: this.transformUserData(error.rawData.user),
          organizations: error.rawData.organizations.map(this.transformOrganizationData),
          cookies: [{
            name: PENDING_SESSION_COOKIE,
            value: error.rawData.pending_authentication_token,
            options: {
              secure: true,
              httpOnly: true,
              maxAge: 60 // user has 60 seconds to select an org before the cookie expires
            }
          }]
        };
      }
      return this.handleError(error);
    }
  }

  // Helper methods for transforming WorkOS types to Unkey types
  private transformUserData(providerUser: any): User {
    return {
      id: providerUser.id,
      orgId: providerUser.organizationId,
      email: providerUser.email,
      firstName: providerUser.firstName,
      lastName: providerUser.lastName,
      avatarUrl: providerUser.profilePictureUrl,
      fullName: providerUser.firstName && providerUser.lastName 
        ? `${providerUser.firstName} ${providerUser.lastName}`
        : null
    };
  }

  private transformOrganizationData(providerOrg: WorkOSOrganization): Organization {
    return {
      id: providerOrg.id,
      name: providerOrg.name,
      createdAt: providerOrg.createdAt,
      updatedAt: providerOrg.updatedAt
    };
  }

  private transformInvitationData(
    providerInvitation: WorkOSInvitation, 
    context: { orgId: string; inviterId?: string }
  ): Invitation {
    return {
      id: providerInvitation.id,
      email: providerInvitation.email,
      state: providerInvitation.state,
      acceptedAt: providerInvitation.acceptedAt,
      revokedAt: providerInvitation.revokedAt,
      expiresAt: providerInvitation.expiresAt,
      token: providerInvitation.token,
      organizationId: providerInvitation.organizationId ?? context.orgId,
      inviterUserId: providerInvitation.inviterUserId ?? context.inviterId,
      createdAt: providerInvitation.createdAt,
      updatedAt: providerInvitation.updatedAt
    };
  }

}
