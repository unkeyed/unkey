import { env } from "@/lib/env";
import {
  AuthenticateWithSessionCookieFailureReason,
  AuthenticationException,
  type AuthenticationResponse,
  GenericServerException,
  OauthException,
  RateLimitExceededException,
  WorkOS,
  type Invitation as WorkOSInvitation,
  type Organization as WorkOSOrganization,
} from "@workos-inc/node";
import { getBaseUrl } from "../utils";
import { BaseAuthProvider } from "./base-provider";
import { getAuthCookieOptions } from "./cookie-security";
import type { Cookie } from "./cookies";
import { getCookie } from "./cookies";
import { getAuth } from "./get-auth";
import {
  AUTH_CHALLENGE_COOKIE,
  type AuthChallengeCookieData,
  type AuthChallengeType,
  AuthErrorCode,
  type AuthErrorResponse,
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  type Membership,
  type MembershipListResponse,
  type MfaEnrollmentStart,
  type MfaFactor,
  type OAuthResult,
  type OrgInviteParams,
  type Organization,
  OrganizationScopeError,
  PENDING_SESSION_COOKIE,
  type PendingAuthChallengeResponse,
  RADAR_ATTEMPT_COOKIE,
  type SessionRefreshResult,
  type SessionValidationResult,
  type SignInViaOAuthOptions,
  UNKEY_SESSION_COOKIE,
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "./types";

// Shape of `rawData.user` on AuthenticationException (snake_case wire format)
type ProviderUser = {
  id: string;
  email: string;
  first_name?: string | null;
  last_name?: string | null;
  profile_picture_url?: string | null;
};

// Pending tokens and challenges are valid for 10 minutes
const PENDING_AUTH_COOKIE_MAX_AGE = 60 * 10;

const CHALLENGE_CODE_BY_TYPE: Record<AuthChallengeType, PendingAuthChallengeResponse["code"]> = {
  mfa: AuthErrorCode.MFA_CHALLENGE_REQUIRED,
  "mfa-enroll": AuthErrorCode.MFA_ENROLLMENT_REQUIRED,
  "radar-email": AuthErrorCode.RADAR_EMAIL_CHALLENGE_REQUIRED,
  "radar-sms": AuthErrorCode.RADAR_SMS_CHALLENGE_REQUIRED,
};

export class WorkOSAuthProvider extends BaseAuthProvider {
  //INFO: Best to leave this alone, some other class might be accessing `instance` implicitly
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
    // iron-webcrypto (which seals the session cookie) rejects shorter
    // passwords at sign-in time with a cryptic "Password string too short"
    // error, so fail fast with an actionable one instead.
    if (cookiePassword.length < 32) {
      throw new Error(
        "WORKOS_COOKIE_PASSWORD must be at least 32 characters. It is used to encrypt the WorkOS session cookie, not a user-facing password.",
      );
    }

    // Initialize properties after validation
    this.clientId = config.clientId;
    this.cookiePassword = cookiePassword; // TypeScript now knows this is string
    this.provider = new WorkOS(config.apiKey, { clientId: config.clientId });

    WorkOSAuthProvider.instance = this;
  }

  /**
   * Builds the cookie set for a successfully established session: the sealed
   * session itself plus removal of any in-flight challenge/Radar state.
   */
  private sessionCookies(sealedSession: string): Cookie[] {
    const expired = { ...getAuthCookieOptions(), maxAge: 0 };
    return [
      {
        name: UNKEY_SESSION_COOKIE,
        value: sealedSession,
        options: { ...getAuthCookieOptions() },
      },
      { name: AUTH_CHALLENGE_COOKIE, value: "", options: expired },
      { name: RADAR_ATTEMPT_COOKIE, value: "", options: expired },
    ];
  }

  private pendingChallengeResponse(
    challenge: AuthChallengeCookieData,
    pendingAuthToken: string,
  ): PendingAuthChallengeResponse {
    const code = CHALLENGE_CODE_BY_TYPE[challenge.type];
    const options = {
      ...getAuthCookieOptions(),
      maxAge: PENDING_AUTH_COOKIE_MAX_AGE,
    };
    return {
      success: false,
      code,
      message: errorMessages[code],
      challengeType: challenge.type,
      cookies: [
        { name: PENDING_SESSION_COOKIE, value: pendingAuthToken, options },
        { name: AUTH_CHALLENGE_COOKIE, value: JSON.stringify(challenge), options },
      ],
    };
  }

  /**
   * Maps an AuthenticationException thrown by any authenticate* call to the
   * pending-state response the UI knows how to continue from. Returns null
   * for errors that are not interruptions (invalid code, expired code, ...).
   */
  private async mapAuthenticationException(error: unknown): Promise<VerificationResult | null> {
    if (!(error instanceof AuthenticationException)) {
      return null;
    }

    const pendingAuthToken = error.pendingAuthenticationToken;
    if (!pendingAuthToken) {
      return null;
    }

    const rawUser = error.rawData.user as ProviderUser | undefined;

    switch (error.code) {
      case "organization_selection_required": {
        if (!rawUser) {
          return null;
        }
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: errorMessages[AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED],
          user: this.transformErrorUserData(rawUser),
          organizations: (error.rawData.organizations ?? []).map((org) => ({
            id: org.id,
            name: org.name,
          })),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: pendingAuthToken,
              options: { ...getAuthCookieOptions() },
            },
          ],
        };
      }

      case "mfa_challenge": {
        // The error lists the user's enrolled factors; create a challenge for
        // the TOTP factor so the client only has to submit the code.
        const factors = (error.rawData.authentication_factors ?? []) as Array<{
          id: string;
          type: string;
        }>;
        const totpFactor = factors.find((factor) => factor.type === "totp");
        if (!totpFactor) {
          return null;
        }
        const challenge = await this.provider.multiFactorAuth.challengeFactor({
          authenticationFactorId: totpFactor.id,
        });
        return this.pendingChallengeResponse(
          { type: "mfa", challengeId: challenge.id },
          pendingAuthToken,
        );
      }

      case "mfa_enrollment": {
        if (!rawUser) {
          return null;
        }
        return this.pendingChallengeResponse(
          { type: "mfa-enroll", userId: rawUser.id, email: rawUser.email },
          pendingAuthToken,
        );
      }

      case "radar_email_challenge": {
        if (!error.radarChallengeId) {
          return null;
        }
        return this.pendingChallengeResponse(
          { type: "radar-email", radarChallengeId: error.radarChallengeId },
          pendingAuthToken,
        );
      }

      case "radar_sms_challenge": {
        if (!rawUser) {
          return null;
        }
        return this.pendingChallengeResponse(
          { type: "radar-sms", userId: rawUser.id },
          pendingAuthToken,
        );
      }

      default:
        return null;
    }
  }

  /**
   * Runs an authenticate* SDK call that should end in a sealed session and
   * maps the outcome to a VerificationResult: session cookies on success, a
   * pending-state response when the attempt was interrupted (org selection,
   * MFA, Radar), and a normalized error otherwise.
   */
  private async completeAuthentication(
    authenticate: () => Promise<AuthenticationResponse>,
  ): Promise<VerificationResult> {
    try {
      const { sealedSession } = await authenticate();

      if (!sealedSession) {
        throw new Error("No sealed session returned");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: this.sessionCookies(sealedSession),
      };
    } catch (error: unknown) {
      return this.mapAuthError(await this.mapAuthenticationException(error), error);
    }
  }

  /**
   * Detects a Radar denial (impossible travel, IP/email blocklist, bot
   * detection, ...). Unlike Radar *challenges*, a block is not a typed error
   * code in the SDK: it comes back from authenticateWith* as a 403
   * OauthException / GenericServerException. Typed challenges (also 403) are
   * resolved earlier via mapAuthenticationException, and bad/expired codes
   * are 400/401, so a remaining 403 here is a denial. We also match an
   * explicit hint in the error body in case the status ever differs.
   */
  private isRadarBlock(error: unknown): boolean {
    // Typed auth exceptions and rate limits both extend GenericServerException
    // — they are handled elsewhere and are never a Radar block.
    if (error instanceof AuthenticationException || error instanceof RateLimitExceededException) {
      return false;
    }

    if (!(error instanceof OauthException || error instanceof GenericServerException)) {
      return false;
    }

    if (error.status === 403) {
      return true;
    }

    const hints = [
      error instanceof OauthException ? error.error : error.code,
      error instanceof OauthException ? error.errorDescription : error.message,
      JSON.stringify(error.rawData ?? ""),
    ]
      .filter(Boolean)
      .join(" ");
    return /radar|blocked|impossible_travel|blocklist|bot_detection/i.test(hints);
  }

  /**
   * Resolves the catch path shared by every authenticate* call: a pending
   * challenge takes priority, then a Radar block gets a clear "contact
   * support" message, then a code-verification failure gets a user-friendly
   * message, otherwise fall back to generic error handling.
   */
  private mapAuthError(pending: VerificationResult | null, error: unknown): VerificationResult {
    if (pending) {
      return pending;
    }
    if (this.isRadarBlock(error)) {
      return this.radarBlockedResponse();
    }
    const codeError = this.mapCodeVerificationError(error);
    if (codeError) {
      return codeError;
    }
    return this.handleError(error as Error);
  }

  /**
   * Maps a one-time-code verification failure (TOTP, magic auth, email
   * verification, Radar email/SMS) to a safe, actionable message. WorkOS
   * phrases these errors for operators — e.g. "One-time code for
   * 'auth_challenge_...' has had too many failed attempts." — which both leaks
   * an internal id and reads as a server error to the user. We match on the
   * message text because the SDK surfaces these as untyped
   * GenericServerException / UnprocessableEntityException without a stable
   * `code`. Returns null when the error is not a recognized code failure so
   * the caller falls through to generic handling.
   */
  private mapCodeVerificationError(error: unknown): AuthErrorResponse | null {
    const message = error instanceof Error ? error.message.toLowerCase() : "";
    if (!message) {
      return null;
    }
    if (message.includes("too many")) {
      return {
        success: false,
        code: AuthErrorCode.RATE_ERROR,
        message: errorMessages[AuthErrorCode.RATE_ERROR],
      };
    }
    if (
      message.includes("incorrect") ||
      message.includes("invalid") ||
      message.includes("expired")
    ) {
      return {
        success: false,
        code: AuthErrorCode.INVALID_CODE,
        message: errorMessages[AuthErrorCode.INVALID_CODE],
      };
    }
    return null;
  }

  /**
   * The single source of truth for a Radar block response, so every flow
   * (email entry, code verification, OAuth) surfaces the same code and the
   * same "contact support" message.
   */
  private radarBlockedResponse(): AuthErrorResponse {
    return {
      success: false,
      code: AuthErrorCode.AUTHENTICATION_BLOCKED,
      message: errorMessages[AuthErrorCode.AUTHENTICATION_BLOCKED],
    };
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
          accessToken: authResult.accessToken,
          userId: authResult.user.id,
          orgId: authResult.organizationId ?? null,
          role: authResult.role ?? null,
          user: this.transformUserData(authResult.user),
        };
      }

      // Only an expired access token is worth a refresh round trip to
      // WorkOS. A missing or undecryptable cookie cannot be refreshed, so
      // skip the doomed API call those used to trigger.
      return {
        isValid: false,
        shouldRefresh: authResult.reason === AuthenticateWithSessionCookieFailureReason.INVALID_JWT,
      };
    } catch (_error) {
      // Since v10 the SDK throws on transient failures (JWKS fetch, network)
      // instead of reporting an invalid session. Attempt a refresh rather
      // than treating the user as signed out because of a blip.
      return { isValid: false, shouldRefresh: true };
    }
  }

  async refreshSession(sessionToken: string | null): Promise<SessionRefreshResult> {
    if (!sessionToken) {
      throw new Error("No session token provided");
    }

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
          accessToken: refreshResult.session.accessToken,
          role: refreshResult.role ?? null,
          user: this.transformUserData(refreshResult.session.user),
        },
        impersonator: refreshResult.session.impersonator,
      };
    }

    throw new Error("reason" in refreshResult ? refreshResult.reason : "Session refresh failed");
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
    } catch (_error) {
      return null;
    }
  }

  async findUser(email: string): Promise<User | null> {
    if (!email) {
      throw new Error("Email address is required.");
    }

    try {
      const { data } = await this.provider.userManagement.listUsers({
        email,
      });
      if (data.length === 0) {
        return null;
      }

      return this.transformUserData(data[0]);
    } catch (_error) {
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

    // Load the current session
    const session = this.provider.userManagement.loadSealedSession({
      sessionData: currentToken,
      cookiePassword: this.cookiePassword,
    });

    const refreshResult = await session.refresh({
      cookiePassword: this.cookiePassword,
      organizationId: newOrgId,
    });

    if (!refreshResult.authenticated || !refreshResult.session || !refreshResult.sealedSession) {
      const errMsg = refreshResult.authenticated ? "" : refreshResult.reason;
      throw new Error(`Organization switch failed ${errMsg}`);
    }

    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + 7);

    return {
      newToken: refreshResult.sealedSession,
      expiresAt,
      session: {
        userId: refreshResult.session.user.id,
        orgId: newOrgId,
        accessToken: refreshResult.session.accessToken,
      },
    };
  }

  // Membership Management
  async listMemberships(userId: string): Promise<MembershipListResponse> {
    try {
      const [user, memberships] = await Promise.all([
        this.getUser(userId),
        this.provider.userManagement.listOrganizationMemberships({
          userId: userId,
          limit: 100,
          statuses: ["active"],
        }),
      ]);

      if (!user) {
        return { data: [], metadata: {} };
      }

      return {
        data: memberships.data.map((membership) => ({
          id: membership.id,
          user,
          // Memberships already carry the organization name, so no getOrg
          // round trip per membership is needed.
          organization: {
            id: membership.organizationId,
            name: membership.organizationName,
          },
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
      // One listUsers call filtered by organization replaces a getUser call
      // per member.
      const [org, members, users] = await Promise.all([
        this.getOrg(orgId),
        this.provider.userManagement.listOrganizationMemberships({
          organizationId: orgId,
          limit: 100,
          statuses: ["active"],
        }),
        this.provider.userManagement.listUsers({
          organizationId: orgId,
          limit: 100,
        }),
      ]);

      const userMap = new Map(users.data.map((user) => [user.id, this.transformUserData(user)]));

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

  /**
   * Asserts the membership belongs to `orgId` before it is mutated. WorkOS
   * scopes membership mutations by ID alone, so without this check an admin of
   * one organization could target a membership ID belonging to another.
   */
  private async assertMembershipInOrg(membershipId: string, orgId: string): Promise<void> {
    const membership = await this.provider.userManagement.getOrganizationMembership(membershipId);
    if (membership.organizationId !== orgId) {
      throw new OrganizationScopeError("membership", membershipId);
    }
  }

  async updateMembership(params: UpdateMembershipParams): Promise<Membership> {
    const { membershipId, role, orgId } = params;
    if (!membershipId || !role || !orgId) {
      throw new Error("Membership id, role, and organization id are required.");
    }

    try {
      await this.assertMembershipInOrg(membershipId, orgId);

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
      if (error instanceof OrganizationScopeError) {
        throw error;
      }
      throw this.handleError(error);
    }
  }

  async removeMembership(membershipId: string, orgId: string): Promise<void> {
    if (!membershipId || !orgId) {
      throw new Error("Membership id and organization id are required.");
    }

    try {
      await this.assertMembershipInOrg(membershipId, orgId);
      await this.provider.userManagement.deleteOrganizationMembership(membershipId);
    } catch (error) {
      if (error instanceof OrganizationScopeError) {
        throw error;
      }
      throw this.handleError(error);
    }
  }

  async deactivateMembership(membershipId: string): Promise<void> {
    if (!membershipId) {
      throw new Error("Membership Id is required");
    }

    try {
      await this.provider.userManagement.deactivateOrganizationMembership(membershipId);
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
    } catch (_error) {
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
    } catch (_error) {
      return null;
    }
  }

  async revokeOrgInvitation(invitationId: string, orgId: string): Promise<void> {
    if (!invitationId || !orgId) {
      throw new Error("Invitation id and organization id are required.");
    }

    try {
      // WorkOS revokes by invitation ID alone, so confirm the invitation
      // belongs to the caller's organization before revoking it.
      const invitation = await this.provider.userManagement.getInvitation(invitationId);
      if (invitation.organizationId !== orgId) {
        throw new OrganizationScopeError("invitation", invitationId);
      }

      await this.provider.userManagement.revokeInvitation(invitationId);
    } catch (error) {
      if (error instanceof OrganizationScopeError) {
        throw error;
      }
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

  /**
   * Sends a magic auth code and returns a success result carrying the Radar
   * attempt cookie so the verification request can be linked to this attempt.
   */
  private async sendMagicAuthCode(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
    radarAuthAttemptId?: string;
  }): Promise<EmailAuthResult> {
    const magicAuth = await this.provider.userManagement.createMagicAuth(params);

    const cookies: Cookie[] = magicAuth.radarAuthAttemptId
      ? [
          {
            name: RADAR_ATTEMPT_COOKIE,
            value: magicAuth.radarAuthAttemptId,
            options: {
              ...getAuthCookieOptions(),
              maxAge: PENDING_AUTH_COOKIE_MAX_AGE,
            },
          },
        ]
      : [];

    return { success: true, cookies };
  }

  async signUpViaEmail(
    params: UserData & {
      ipAddress?: string;
      userAgent?: string;
    },
  ): Promise<EmailAuthResult> {
    const { email, firstName, lastName, ipAddress, userAgent } = params;

    try {
      // Create the user with WorkOS. Passing the request metadata enrolls
      // this request with Radar, which links the attempt to the
      // authentication that follows.
      const createUserResponse = await this.provider.userManagement.createUser({
        firstName,
        lastName,
        email,
        ipAddress,
        userAgent,
      });

      return await this.sendMagicAuthCode({
        email,
        ipAddress,
        userAgent,
        radarAuthAttemptId: createUserResponse.radarAuthAttemptId,
      });
    } catch (error: unknown) {
      // Radar can deny at attempt creation (e.g. a blocked country), so
      // surface the unified block message here too.
      if (this.isRadarBlock(error)) {
        return this.radarBlockedResponse();
      }
      // Perform structural narrowing before accessing properties
      if (
        typeof error === "object" &&
        error !== null &&
        "errors" in error &&
        Array.isArray(error.errors) &&
        error.errors.some((detail: { code?: string }) => detail.code === "email_not_available")
      ) {
        // The email is taken. If its owner never finished verifying, this is
        // someone retrying an abandoned sign-up: resend the code so they can
        // complete it instead of being locked out with "already registered".
        try {
          const { data } = await this.provider.userManagement.listUsers({ email });
          const existingUser = data[0];
          if (existingUser && !existingUser.emailVerified) {
            return await this.sendMagicAuthCode({ email, ipAddress, userAgent });
          }
        } catch (_lookupError) {
          // Fall through to the duplicate-email error below
        }
        return this.handleError(new Error(AuthErrorCode.EMAIL_ALREADY_EXISTS));
      }
      if (
        typeof error === "object" &&
        error !== null &&
        "message" in error &&
        typeof error.message === "string" &&
        error.message.includes("email_required")
      ) {
        return this.handleError(new Error(AuthErrorCode.INVALID_EMAIL));
      }
      if (
        typeof error === "object" &&
        error !== null &&
        "code" in error &&
        error.code === "user_creation_error"
      ) {
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

    try {
      const { data } = await this.provider.userManagement.listUsers({ email });

      if (data.length === 0) {
        return this.handleError(new Error(AuthErrorCode.ACCOUNT_NOT_FOUND));
      }

      return await this.sendMagicAuthCode({ email, ipAddress, userAgent });
    } catch (error) {
      if (this.isRadarBlock(error)) {
        return this.radarBlockedResponse();
      }
      return this.handleError(error);
    }
  }

  async resendAuthCode(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
    radarAuthAttemptId?: string;
  }): Promise<EmailAuthResult> {
    try {
      return await this.sendMagicAuthCode(params);
    } catch (error) {
      return this.handleError(error);
    }
  }

  async verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
    ipAddress?: string;
    userAgent?: string;
    radarAuthAttemptId?: string;
  }): Promise<VerificationResult> {
    const { email, code, invitationToken, ipAddress, userAgent, radarAuthAttemptId } = params;

    return this.completeAuthentication(() =>
      this.provider.userManagement.authenticateWithMagicAuth({
        clientId: this.clientId,
        code,
        email,
        invitationToken,
        ipAddress,
        userAgent,
        radarAuthAttemptId,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      }),
    );
  }

  async verifyEmail(params: {
    code: string;
    token: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    const { code, token, ipAddress, userAgent } = params;

    return this.completeAuthentication(() =>
      this.provider.userManagement.authenticateWithEmailVerification({
        clientId: this.clientId,
        code,
        pendingAuthenticationToken: token,
        ipAddress,
        userAgent,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      }),
    );
  }

  async completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult> {
    const result = await this.completeAuthentication(() =>
      this.provider.userManagement.authenticateWithOrganizationSelection({
        pendingAuthenticationToken: params.pendingAuthToken,
        organizationId: params.orgId,
        clientId: this.clientId,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      }),
    );

    // WorkOS rejects a stale pending token with "user does not belong to
    // <org>"; surface that as an expired pending session instead.
    if (!result.success && result.message.includes("does not belong to")) {
      return {
        success: false,
        code: AuthErrorCode.PENDING_SESSION_EXPIRED,
        message: errorMessages[AuthErrorCode.PENDING_SESSION_EXPIRED],
      };
    }

    return result;
  }

  // MFA and Radar challenge completion

  async completeMfaChallenge(params: {
    code: string;
    challengeId: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    return this.completeAuthentication(() =>
      this.provider.userManagement.authenticateWithTotp({
        clientId: this.clientId,
        code: params.code,
        authenticationChallengeId: params.challengeId,
        pendingAuthenticationToken: params.pendingAuthToken,
        ipAddress: params.ipAddress,
        userAgent: params.userAgent,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      }),
    );
  }

  async beginMfaEnrollment(params: {
    userId: string;
    email: string;
  }): Promise<MfaEnrollmentStart> {
    // Clean up orphans from abandoned enrollments so they don't accumulate in
    // WorkOS. An enrolled-but-unverified factor comes back from the list
    // without its `totp` metadata; a verified, in-use factor always has it.
    // Delete only the former so we never remove a factor the user still relies
    // on. Best-effort: a cleanup failure must not block a fresh enrollment.
    try {
      const existing = await this.provider.multiFactorAuth.listUserAuthFactors({
        userId: params.userId,
        limit: 100,
      });
      await Promise.all(
        existing.data
          .filter((factor) => !factor.totp)
          .map((factor) => this.provider.multiFactorAuth.deleteFactor(factor.id)),
      );
    } catch (error) {
      console.error("Failed to clean up pending MFA factors:", error);
    }

    const { authenticationFactor, authenticationChallenge } =
      await this.provider.multiFactorAuth.createUserAuthFactor({
        userId: params.userId,
        type: "totp",
        totpIssuer: "Unkey",
        totpUser: params.email,
      });

    return {
      factorId: authenticationFactor.id,
      challengeId: authenticationChallenge.id,
      qrCode: authenticationFactor.totp.qrCode,
      secret: authenticationFactor.totp.secret,
      uri: authenticationFactor.totp.uri,
    };
  }

  async verifyMfaEnrollment(params: {
    challengeId: string;
    code: string;
  }): Promise<boolean> {
    const response = await this.provider.multiFactorAuth.verifyChallenge({
      authenticationChallengeId: params.challengeId,
      code: params.code,
    });
    return response.valid;
  }

  async listMfaFactors(userId: string): Promise<MfaFactor[]> {
    const factors = await this.provider.multiFactorAuth.listUserAuthFactors({
      userId,
      limit: 100,
    });

    // Surface only verified, usable factors. An enrolled-but-unverified factor
    // comes back without its `totp` metadata; showing it would both mislead
    // ("MFA configured" when it isn't) and, in settings, hide the enroll
    // button behind an unusable orphan.
    return factors.data
      .filter((factor) => factor.totp)
      .map((factor) => ({
        id: factor.id,
        type: factor.type,
        issuer: factor.totp?.issuer ?? "",
        user: factor.totp?.user ?? "",
        createdAt: factor.createdAt,
      }));
  }

  async removeMfaFactor(factorId: string): Promise<void> {
    await this.provider.multiFactorAuth.deleteFactor(factorId);
  }

  async completeRadarEmailChallenge(params: {
    code: string;
    radarChallengeId: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    return this.completeAuthentication(() =>
      this.provider.userManagement.authenticateWithRadarEmailChallenge({
        clientId: this.clientId,
        code: params.code,
        radarChallengeId: params.radarChallengeId,
        pendingAuthenticationToken: params.pendingAuthToken,
        ipAddress: params.ipAddress,
        userAgent: params.userAgent,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      }),
    );
  }

  async sendRadarSmsCode(params: {
    userId: string;
    phoneNumber: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<{ verificationId: string; phoneNumber: string }> {
    const response = await this.provider.userManagement.sendRadarSmsChallenge({
      userId: params.userId,
      pendingAuthenticationToken: params.pendingAuthToken,
      phoneNumber: params.phoneNumber,
      ipAddress: params.ipAddress,
      userAgent: params.userAgent,
    });

    return {
      verificationId: response.verificationId,
      phoneNumber: response.phoneNumber,
    };
  }

  async completeRadarSmsChallenge(params: {
    code: string;
    verificationId: string;
    phoneNumber: string;
    pendingAuthToken: string;
    ipAddress?: string;
    userAgent?: string;
  }): Promise<VerificationResult> {
    return this.completeAuthentication(() =>
      this.provider.userManagement.authenticateWithRadarSmsChallenge({
        clientId: this.clientId,
        code: params.code,
        verificationId: params.verificationId,
        phoneNumber: params.phoneNumber,
        pendingAuthenticationToken: params.pendingAuthToken,
        ipAddress: params.ipAddress,
        userAgent: params.userAgent,
        session: {
          sealSession: true,
          cookiePassword: this.cookiePassword,
        },
      }),
    );
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
    } catch (_error) {
      return null;
    }
  }

  async revokeSession(sessionToken: string): Promise<void> {
    if (!sessionToken) {
      return;
    }

    try {
      const session = this.provider.userManagement.loadSealedSession({
        sessionData: sessionToken,
        cookiePassword: this.cookiePassword,
      });

      const authResult = await session.authenticate();
      if (!authResult.authenticated) {
        return;
      }

      await this.provider.userManagement.revokeSession({
        sessionId: authResult.sessionId,
      });
    } catch (_error) {
      // Swallow revoke failures so logout can always complete client-side.
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
        ipAddress:
          callbackRequest.headers.get("x-forwarded-for")?.split(",")[0].trim() || undefined,
        userAgent: callbackRequest.headers.get("user-agent") || undefined,
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
        cookies: this.sessionCookies(sealedSession),
      };
    } catch (error: unknown) {
      if (
        error instanceof AuthenticationException &&
        error.code === "email_verification_required" &&
        error.pendingAuthenticationToken
      ) {
        return {
          success: false,
          code: AuthErrorCode.EMAIL_VERIFICATION_REQUIRED,
          message: errorMessages[AuthErrorCode.EMAIL_VERIFICATION_REQUIRED],
          user: this.transformErrorUserData({
            id: "UNKNOWN", // WorkOS doesn't return a user id in this scenario, and its the ONLY scenario where there isn't one available. Easier to just pass a string than to make the unkey User id nullable
            email: (error.rawData.email as string | undefined) || "",
          }),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: error.pendingAuthenticationToken,
              options: { ...getAuthCookieOptions(), maxAge: PENDING_AUTH_COOKIE_MAX_AGE },
            },
          ],
        };
      }

      return this.mapAuthError(await this.mapAuthenticationException(error), error);
    }
  }

  // Helper methods for transforming WorkOS types to Unkey types
  private transformErrorUserData(providerUser: ProviderUser): User {
    const firstName = providerUser.first_name ?? null;
    const lastName = providerUser.last_name ?? null;
    return {
      id: providerUser.id,
      email: providerUser.email,
      firstName,
      lastName,
      avatarUrl: providerUser.profile_picture_url ?? null,
      fullName: firstName && lastName ? `${firstName} ${lastName}` : null,
    };
  }

  private transformUserData(providerUser: {
    id: string;
    email: string;
    firstName: string | null;
    lastName: string | null;
    profilePictureUrl: string | null;
  }): User {
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
