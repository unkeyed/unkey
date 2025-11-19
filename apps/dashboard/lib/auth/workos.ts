import { env } from "@/lib/env";
import {
  WorkOS,
  type Invitation as WorkOSInvitation,
  type Organization as WorkOSOrganization,
} from "@workos-inc/node";
import { getBaseUrl } from "../utils";
import { BaseAuthProvider } from "./base-provider";
import { getAuthCookieOptions } from "./cookie-security";
import { getCookie } from "./cookies";
import { getAuth } from "./get-auth";
import {
  AuthErrorCode,
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  type Membership,
  type MembershipListResponse,
  type OAuthResult,
  type OrgInviteParams,
  type Organization,
  PENDING_SESSION_COOKIE,
  type SessionRefreshResult,
  type SessionValidationResult,
  type SignInViaOAuthOptions,
  UNKEY_SESSION_COOKIE,
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
  type UserData,
  type VerificationResult,
  WORKOS_RADAR_API_URL,
} from "./types";

type WorkOSErrorCode =
  | "organization_selection_required"
  | "email_verification_required"
  | "email_not_available"
  | "user_creation_error"
  | "invalid_code"
  | "expired_code"
  | "invalid_invitation"
  | "missing_required_fields";

type WorkOSError = {
  errors?: {
    code: string;
    message?: string;
  }[];
  message: string;
  code?: string;
};

type WorkOSAuthError = {
  message: string;
  rawData?: {
    code: WorkOSErrorCode;
    message: string;
    user: ProviderUser;
    organizations: WorkOSOrganization[];
    pending_authentication_token: string;
    email?: string;
  };
};

type ProviderUser = {
  id: string;
  email: string;
  firstName: string | null;
  lastName: string | null;
  profilePictureUrl: string | null;
};

type Decision = {
  action: "allow" | "block" | "challenge";
  reason?: string;
};

export class WorkOSAuthProvider extends BaseAuthProvider {
  //INFO: Best to leave this alone, some other class might be accessing `instance` implicitly
  private static instance: WorkOSAuthProvider | null = null;
  private readonly provider: WorkOS;
  private readonly apiKey: string;
  private readonly clientId: string;
  private readonly cookiePassword: string;

  constructor(config: { apiKey: string; clientId: string }) {
    super();

    const cookiePassword = env().WORKOS_COOKIE_PASSWORD;
    if (!cookiePassword) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required for WorkOS authentication");
    }

    // Initialize properties after validation
    this.apiKey = config.apiKey;
    this.clientId = config.clientId;
    this.cookiePassword = cookiePassword; // TypeScript now knows this is string
    this.provider = new WorkOS(config.apiKey, { clientId: config.clientId });

    WorkOSAuthProvider.instance = this;
  }

  private async checkRadar(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
    auth_method?: string;
    action?: string;
  }): Promise<Decision> {
    try {
      const response = await fetch(WORKOS_RADAR_API_URL, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${this.apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email: params.email,
          ip_address: params.ipAddress,
          user_agent: params.userAgent,
          auth_method: params.auth_method,
          action: params.action,
        }),
      });

      if (!response.ok) {
        console.error("Radar API error:", response.status);
        return { action: "allow" };
      }

      const data = await response.json();
      return {
        action: data.action || "allow",
        reason: data.reason,
      };
    } catch (error) {
      console.error("Failed to check Radar:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return { action: "allow" };
    }
  }

  // Session Management
  async validateSession(sessionToken: string): Promise<SessionValidationResult> {
    if (!sessionToken) {
      return { isValid: false, shouldRefresh: false };
    }

    try {
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: sessionToken,
        cookiePassword: this.cookiePassword,
      });

      const authResult = await session.authenticate();

      if (authResult.authenticated) {
        return {
          isValid: true,
          shouldRefresh: false,
          impersonator: authResult.impersonator,
          userId: authResult.user.id,
          orgId: authResult.organizationId ?? null,
          role: authResult.role ?? null,
        };
      }

      return { isValid: false, shouldRefresh: true };
    } catch (error) {
      console.error("Session validation error:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return { isValid: false, shouldRefresh: false };
    }
  }

  async refreshSession(sessionToken: string | null): Promise<SessionRefreshResult> {
    if (!sessionToken) {
      throw new Error("No session token provided");
    }

    try {
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: sessionToken,
        cookiePassword: this.cookiePassword,
      });

      const refreshResult = await session.refresh({
        cookiePassword: this.cookiePassword,
      });

      if (refreshResult.authenticated && refreshResult.session) {
        // Set expiration to 7 days from now
        const expiresAt = new Date();
        expiresAt.setDate(expiresAt.getDate() + 7);

        if (!refreshResult.sealedSession) {
          throw new Error("Session refresh failed due to missing sealedSession");
        }

        return {
          newToken: refreshResult.sealedSession,
          expiresAt,
          session: {
            userId: refreshResult.session.user.id,
            orgId: refreshResult.session.organizationId ?? null,
            role: refreshResult.role ?? null,
          },
          impersonator: refreshResult.session.impersonator,
        };
      }

      throw new Error("reason" in refreshResult ? refreshResult.reason : "Session refresh failed");
    } catch (error) {
      console.error("Session refresh error:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      throw error;
    }
  }

  // User Management
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
      console.error("Failed to get user:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return null;
    }
  }

  async findUser(email: string): Promise<User | null> {
    if (!email) {
      throw new Error("Email address is required.");
    }

    try {
      const user = await this.provider.userManagement.listUsers({
        email,
      });
      if (!user) {
        return null;
      }

      return this.transformUserData(user.data[0]);
    } catch (error) {
      console.error("Failed to find user:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return null;
    }
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

    try {
      const org = await this.createOrg(name);
      const membership = await this.provider.userManagement.createOrganizationMembership({
        organizationId: org.id,
        userId,
        roleSlug: "admin",
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
      const org = await this.provider.organizations.createOrganization({
        name,
      });
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
        name,
      });
      return this.transformOrganizationData(org);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async switchOrg(newOrgId: string): Promise<SessionRefreshResult> {
    // Get current session token
    const currentToken = await getCookie(UNKEY_SESSION_COOKIE);
    if (!currentToken) {
      throw new Error("No active session found");
    }

    try {
      // Load the current session
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: currentToken,
        cookiePassword: this.cookiePassword,
      });

      // Create a new session with the new organization ID
      const refreshResult = await session.refresh({
        cookiePassword: this.cookiePassword,
        organizationId: newOrgId,
      });

      if (!refreshResult.authenticated || !refreshResult.session || !refreshResult.sealedSession) {
        const errMsg = refreshResult.authenticated ? "" : refreshResult.reason;
        throw new Error(`Organization switch failed ${errMsg}`);
      }

      // Set expiration to 7 days from now
      const expiresAt = new Date();
      expiresAt.setDate(expiresAt.getDate() + 7);

      return {
        newToken: refreshResult.sealedSession,
        expiresAt,
        session: {
          userId: refreshResult.session.user.id,
          orgId: newOrgId,
        },
      };
    } catch (error) {
      console.error("Organization switch error:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      throw error;
    }
  }

  // Membership Management
  async listMemberships(userId: string): Promise<MembershipListResponse> {
    try {
      const user = await this.getUser(userId);

      if (!user) {
        return { data: [], metadata: {} };
      }

      const memberships = await this.provider.userManagement.listOrganizationMemberships({
        userId: userId,
        limit: 100,
        statuses: ["active"],
      });

      // Fetch organizations for each membership
      const orgs = await Promise.all(memberships.data.map((m) => this.getOrg(m.organizationId)));

      const orgMap = new Map(orgs.map((org) => [org.id, org]));

      return {
        data: memberships.data.map((membership) => ({
          id: membership.id,
          user,
          // biome-ignore lint/style/noNonNullAssertion: Safe to leave
          organization: orgMap.get(membership.organizationId)!,
          role: membership.role.slug,
          createdAt: membership.createdAt,
          updatedAt: membership.updatedAt,
          status: membership.status,
        })),
        metadata: memberships.listMetadata || {},
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
        statuses: ["active"],
      });

      // Get user data for each member
      const users = await Promise.all(members.data.map((m) => this.getUser(m.userId)));

      // Create user map excluding null results
      const userMap = new Map(
        users.filter((user): user is User => user !== null).map((user) => [user.id, user]),
      );

      return {
        data: members.data.map((member) => {
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
            status: member.status,
          };
        }),
        metadata: members.listMetadata || {},
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
        { roleSlug: role },
      );

      // Get related data
      const [org, user] = await Promise.all([
        this.getOrg(membership.organizationId),
        this.getUser(membership.userId),
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
        status: membership.status,
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
      const { userId } = await getAuth();
      if (!userId) {
        throw new Error("User must be authenticated to invite members.");
      }

      const invitation = await this.provider.userManagement.sendInvitation({
        email,
        organizationId: orgId,
        roleSlug: role,
        inviterUserId: userId,
      });

      return this.transformInvitationData(invitation, {
        orgId,
        inviterId: userId,
      });
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
        organizationId: orgId,
      });

      return {
        data: invitationsList.data
          .map((invitation) => this.transformInvitationData(invitation, { orgId }))
          .filter((invitation) => invitation.state === "pending" || invitation.state === "expired"),
        metadata: invitationsList.listMetadata || {},
      };
    } catch (error) {
      console.error("Failed to get organization invitations list:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return {
        data: [],
        metadata: {},
      };
    }
  }

  async getInvitation(invitationToken: string): Promise<Invitation | null> {
    if (!invitationToken) {
      return null;
    }

    try {
      const invitation = await this.provider.userManagement.findInvitationByToken(invitationToken);

      return this.transformInvitationData(invitation);
    } catch (error) {
      console.error("Error retrieving invitation:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return null;
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

  async acceptInvitation(invitationId: string): Promise<Invitation> {
    if (!invitationId) {
      throw new Error("Invitation Id is required");
    }

    try {
      const invitation = await this.provider.userManagement.acceptInvitation(invitationId);
      return this.transformInvitationData(invitation);
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // Authentication Management

  async signUpViaEmail(
    params: UserData & { ipAddress?: string; userAgent?: string },
  ): Promise<EmailAuthResult> {
    const { email, firstName, lastName, ipAddress, userAgent } = params;
    const auth_method = "Email_OTP"; // WorkOS value
    const action = "sign-up"; // WorkOS value
    // Check Radar before proceeding with signup
    const radarDecision = await this.checkRadar({
      email,
      ipAddress,
      userAgent,
      auth_method,
      action,
    });

    // Right now, challenge is treated as a block until we implement a captcha or challenge mechanism
    // Worst case we get a support request and manually allow
    // initial radar testing shows that bots are likely to be challenged than blocked
    if (radarDecision.action !== "allow") {
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: radarDecision.reason || "Sign up blocked due to suspicious activity",
      };
    }

    try {
      // Create the user with WorkOS
      await this.provider.userManagement.createUser({
        firstName,
        lastName,
        email,
      });

      // Send magic auth email
      await this.provider.userManagement.createMagicAuth({ email });

      return { success: true };
    } catch (error: unknown) {
      const workosError = error as WorkOSError;

      if (workosError.errors?.some((detail) => detail.code === "email_not_available")) {
        return this.handleError(new Error(AuthErrorCode.EMAIL_ALREADY_EXISTS));
      }
      if (workosError.message?.includes("email_required")) {
        return this.handleError(new Error(AuthErrorCode.INVALID_EMAIL));
      }
      if (workosError.code === "user_creation_error") {
        return this.handleError(new Error(AuthErrorCode.USER_CREATION_FAILED));
      }
      return this.handleError(error as Error);
    }
  }

  async signInViaEmail(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<EmailAuthResult> {
    const { email, ipAddress, userAgent } = params;
    const auth_method = "Email_OTP"; // WorkOS value
    const action = "sign-in"; // WorkOS value
    // Check Radar before proceeding with signin
    const radarDecision = await this.checkRadar({
      email,
      ipAddress,
      userAgent,
      auth_method,
      action,
    });

    // Right now, challenge is treated as a block until we implement a captcha or challenge mechanism
    // Worst case we get a support request and manually allow
    // initial radar testing shows that bots are likely to be challenged than blocked
    if (radarDecision.action !== "allow") {
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: radarDecision.reason || "Sign in blocked due to suspicious activity",
      };
    }

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

  async verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
  }): Promise<VerificationResult> {
    const { email, code, invitationToken } = params;

    try {
      const { sealedSession } = await this.provider.userManagement.authenticateWithMagicAuth({
        clientId: this.clientId,
        code,
        email,
        invitationToken,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      });

      if (!sealedSession) {
        throw new Error("No sealed session returned");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: UNKEY_SESSION_COOKIE,
            value: sealedSession,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      const authError = error as WorkOSAuthError;

      // Handle organization selection required case
      if (authError.rawData?.code === "organization_selection_required") {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: authError.rawData.message,
          user: this.transformUserData(authError.rawData.user),
          organizations: authError.rawData.organizations.map(this.transformOrganizationData),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: authError.rawData.pending_authentication_token,
              options: { ...getAuthCookieOptions() },
            },
          ],
        };
      }
      return this.handleError(error as Error);
    }
  }

  async verifyEmail(params: {
    code: string;
    token: string;
  }): Promise<VerificationResult> {
    const { code, token } = params;

    try {
      const { sealedSession } =
        await this.provider.userManagement.authenticateWithEmailVerification({
          clientId: this.clientId,
          code,
          pendingAuthenticationToken: token,
          session: {
            sealSession: true,
            cookiePassword: this.cookiePassword,
          },
        });

      if (!sealedSession) {
        throw new Error("No sealed session returned");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: UNKEY_SESSION_COOKIE,
            value: sealedSession,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      // Handle organization selection required case
      console.error("Email verification error:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });

      const authError = error as WorkOSAuthError;

      if (authError.rawData?.code === "organization_selection_required") {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: authError.rawData.message,
          user: this.transformUserData(authError.rawData.user),
          organizations: authError.rawData.organizations.map(this.transformOrganizationData),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: authError.rawData.pending_authentication_token,
              options: { ...getAuthCookieOptions() },
            },
          ],
        };
      }
      return this.handleError(error as Error);
    }
  }

  async completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult> {
    try {
      const { sealedSession } =
        await this.provider.userManagement.authenticateWithOrganizationSelection({
          pendingAuthenticationToken: params.pendingAuthToken,
          organizationId: params.orgId,
          clientId: this.clientId,
          session: {
            sealSession: true,
            cookiePassword: this.cookiePassword,
          },
        });

      if (!sealedSession) {
        throw new Error("No sealed session returned");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: UNKEY_SESSION_COOKIE,
            value: sealedSession,
            options: { ...getAuthCookieOptions() },
          },
        ],
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
        cookiePassword: this.cookiePassword,
      });

      return await session.getLogoutUrl();
    } catch (error) {
      console.error("Failed to get sign out URL:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return null;
    }
  }

  // OAuth Methods
  signInViaOAuth(options: SignInViaOAuthOptions): string {
    const { provider, redirectUrlComplete } = options;
    const state = encodeURIComponent(JSON.stringify({ redirectUrlComplete }));
    const baseUrl = getBaseUrl();
    const redirect = `${baseUrl}/auth/sso-callback`;
    return this.provider.userManagement.getAuthorizationUrl({
      clientId: this.clientId,
      redirectUri: env().NEXT_PUBLIC_WORKOS_REDIRECT_URI ?? redirect,
      provider: provider === "github" ? "GitHubOAuth" : "GoogleOAuth",
      state,
    });
  }

  async completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult> {
    const url = new URL(callbackRequest.url);
    const code = url.searchParams.get("code");
    const state = url.searchParams.get("state");

    if (!code) {
      return this.handleError(new Error(AuthErrorCode.MISSING_REQUIRED_FIELDS));
    }

    try {
      const { sealedSession } = await this.provider.userManagement.authenticateWithCode({
        clientId: this.clientId,
        code,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      });

      if (!sealedSession) {
        throw new Error("No sealed session returned");
      }

      const redirectUrlComplete = state
        ? JSON.parse(decodeURIComponent(state)).redirectUrlComplete
        : "/apis";

      return {
        success: true,
        redirectTo: redirectUrlComplete,
        cookies: [
          {
            name: UNKEY_SESSION_COOKIE,
            value: sealedSession,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      const authError = error as WorkOSAuthError;

      if (authError.rawData?.code === "organization_selection_required") {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: authError.rawData.message,
          user: this.transformUserData(authError.rawData.user),
          organizations: authError.rawData.organizations?.map(this.transformOrganizationData) || [],
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: authError.rawData.pending_authentication_token,
              options: { ...getAuthCookieOptions(), maxAge: 60 },
            },
          ],
        };
      }

      if (authError.rawData?.code === "email_verification_required") {
        return {
          success: false,
          code: AuthErrorCode.EMAIL_VERIFICATION_REQUIRED,
          message: authError.message,
          user: this.transformUserData({
            id: "UNKNOWN", // WorkOS doesn't return a user id in this scenario, and its the ONLY scenario where there isn't one available. Easier to just pass a string than to make the unkey User id nullable
            email: authError.rawData.email || "",
            firstName: null,
            lastName: null,
            profilePictureUrl: null,
          }),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: authError.rawData.pending_authentication_token,
              options: { ...getAuthCookieOptions(), maxAge: 60 * 10 },
            },
          ],
        };
      }

      return this.handleError(error as Error);
    }
  }

  // Helper methods for transforming WorkOS types to Unkey types
  private transformUserData(providerUser: ProviderUser): User {
    return {
      id: providerUser.id,
      email: providerUser.email,
      firstName: providerUser.firstName,
      lastName: providerUser.lastName,
      avatarUrl: providerUser.profilePictureUrl,
      fullName:
        providerUser.firstName && providerUser.lastName
          ? `${providerUser.firstName} ${providerUser.lastName}`
          : null,
    };
  }

  private transformOrganizationData(providerOrg: WorkOSOrganization): Organization {
    return {
      id: providerOrg.id,
      name: providerOrg.name,
      createdAt: providerOrg.createdAt,
      updatedAt: providerOrg.updatedAt,
    };
  }

  private transformInvitationData(
    providerInvitation: WorkOSInvitation,
    context?: { orgId: string; inviterId?: string },
  ): Invitation {
    return {
      id: providerInvitation.id,
      email: providerInvitation.email,
      state: providerInvitation.state,
      acceptedAt: providerInvitation.acceptedAt,
      revokedAt: providerInvitation.revokedAt,
      expiresAt: providerInvitation.expiresAt,
      token: providerInvitation.token,
      organizationId: providerInvitation.organizationId ?? context?.orgId,
      inviterUserId: providerInvitation.inviterUserId ?? context?.inviterId,
      createdAt: providerInvitation.createdAt,
      updatedAt: providerInvitation.updatedAt,
    };
  }
}
