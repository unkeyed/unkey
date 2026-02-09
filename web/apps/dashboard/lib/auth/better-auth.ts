import { env } from "@/lib/env";
import { APIError } from "better-auth/api";
import { BaseAuthProvider } from "./base-provider";
import { type BetterAuthInstance, getBetterAuthInstance } from "./better-auth-server";
import {
  type BetterAuthInvitation,
  type BetterAuthMember,
  type BetterAuthOrganization,
  type BetterAuthUser,
  transformInvitation,
  transformMembership,
  transformOrganization,
  transformUser,
} from "./better-auth-transformers";
import { getAuthCookieOptions } from "./cookie-security";
import { getCookie } from "./cookies";
import {
  AuthErrorCode,
  type AuthErrorResponse,
  BETTER_AUTH_SESSION_COOKIE,
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
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
  type UserData,
  type VerificationResult,
} from "./types";

/**
 * Bot detection decision result
 */
type BotDecision = "allow" | "challenge" | "block";

/**
 * Builds the cookie header for Better Auth API calls.
 * Better Auth expects session tokens in the cookie header, not as Bearer tokens.
 *
 * @param sessionToken - The session token to include in the cookie
 * @returns Headers object with the cookie set
 */
function buildSessionHeaders(sessionToken: string): { cookie: string } {
  return { cookie: `better-auth.session_token=${sessionToken}` };
}

/**
 * BetterAuthProvider
 *
 * Authentication provider implementation using Better Auth library.
 * Implements all abstract methods from BaseAuthProvider by delegating
 * to the Better Auth server-side API.
 *
 * Requirements: 1.2, 1.3
 */
export class BetterAuthProvider extends BaseAuthProvider {
  private readonly auth: BetterAuthInstance;
  private readonly turnstileSecretKey: string | undefined;

  constructor() {
    super();

    // Validate required environment variables
    const config = env();
    if (!config.BETTER_AUTH_SECRET) {
      throw new Error("BETTER_AUTH_SECRET is required when AUTH_PROVIDER=better-auth");
    }
    if (!config.BETTER_AUTH_URL) {
      throw new Error("BETTER_AUTH_URL is required when AUTH_PROVIDER=better-auth");
    }

    this.auth = getBetterAuthInstance();
    this.turnstileSecretKey = config.CLOUDFLARE_TURNSTILE_SECRET_KEY;
  }

  /**
   * Checks for bot risk based on email, IP address, and user agent.
   * Returns a decision on whether to allow, challenge, or block the request.
   *
   * Requirements: 8.1, 8.4, 8.5
   *
   * @param params - Parameters for bot detection evaluation
   * @returns Decision: "allow", "challenge", or "block"
   */
  private checkBotRisk(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
  }): BotDecision {
    // Simple heuristic-based bot detection
    // In production, this could call an external service like Cloudflare Turnstile
    // or implement more sophisticated detection logic

    const { userAgent } = params;

    // Block known bot user agents
    if (userAgent) {
      const lowerUA = userAgent.toLowerCase();
      const botPatterns = [
        "bot",
        "crawler",
        "spider",
        "scraper",
        "curl",
        "wget",
        "python-requests",
        "go-http-client",
      ];

      for (const pattern of botPatterns) {
        if (lowerUA.includes(pattern)) {
          return "block";
        }
      }
    }

    // Default to allow - fail-open for better UX
    // The actual Turnstile verification happens in the server action layer
    return "allow";
  }

  /**
   * Maps Better Auth errors to AuthErrorResponse.
   * Translates Better Auth-specific error codes to the Unkey AuthErrorCode enum.
   *
   * Requirements: 11.1, 11.2, 11.3
   *
   * @param error - The error from Better Auth API
   * @returns Standardized AuthErrorResponse
   */
  private mapBetterAuthError(error: unknown): AuthErrorResponse {
    if (error instanceof APIError) {
      const code = error.body?.code ?? error.message;

      switch (code) {
        case "USER_ALREADY_EXISTS":
        case "user_already_exists":
          return this.handleError(new Error(AuthErrorCode.EMAIL_ALREADY_EXISTS));

        case "INVALID_OTP":
        case "OTP_EXPIRED":
        case "invalid_otp":
        case "otp_expired":
        case "TOO_MANY_ATTEMPTS":
        case "too_many_attempts":
          return this.handleError(new Error(AuthErrorCode.UNKNOWN_ERROR));

        case "USER_NOT_FOUND":
        case "user_not_found":
          return this.handleError(new Error(AuthErrorCode.ACCOUNT_NOT_FOUND));

        case "MISSING_FIELDS":
        case "missing_fields":
          return this.handleError(new Error(AuthErrorCode.MISSING_REQUIRED_FIELDS));

        case "RATE_LIMIT_EXCEEDED":
        case "rate_limit_exceeded":
          return this.handleError(new Error(AuthErrorCode.RATE_ERROR));

        case "NETWORK_ERROR":
        case "network_error":
        case "FETCH_ERROR":
        case "fetch_error":
          return this.handleError(new Error(AuthErrorCode.NETWORK_ERROR));

        case "PENDING_SESSION_EXPIRED":
        case "pending_session_expired":
          return this.handleError(new Error(AuthErrorCode.PENDING_SESSION_EXPIRED));

        default:
          return this.handleError(error);
      }
    }

    if (error instanceof Error) {
      return this.handleError(error);
    }

    return this.handleError(new Error("Unknown error"));
  }

  // ============================================================================
  // Email Authentication Methods (Task 4.2)
  // ============================================================================

  /**
   * Creates a new user account and sends an OTP code to the provided email.
   *
   * Requirements: 2.1, 8.1, 8.4
   *
   * @param params - User data including email, first name, last name, and optional request metadata
   * @returns Result of the sign-up attempt
   */
  async signUpViaEmail(
    params: UserData & {
      ipAddress?: string;
      userAgent?: string;
      bypassRadar?: boolean;
    },
  ): Promise<EmailAuthResult> {
    const { email, firstName, lastName, ipAddress, userAgent, bypassRadar } = params;

    // Bot detection check (unless bypassed after Turnstile verification)
    if (!bypassRadar) {
      const botDecision = this.checkBotRisk({ email, ipAddress, userAgent });

      if (botDecision === "challenge") {
        return {
          success: false,
          code: AuthErrorCode.RADAR_CHALLENGE_REQUIRED,
          message: "Please complete the verification challenge to continue.",
          email,
          challengeParams: {
            ipAddress,
            userAgent,
            authMethod: "Email_OTP",
            action: "sign-up",
          },
        };
      }

      if (botDecision === "block") {
        return this.handleError(new Error(AuthErrorCode.RADAR_BLOCKED));
      }
    }

    try {
      // Create the user with Better Auth
      const fullName = `${firstName} ${lastName}`.trim();
      await this.auth.api.signUpEmail({
        body: {
          email,
          name: fullName,
          password: crypto.randomUUID(), // Better Auth requires a password; we use OTP for auth
        },
      });

      // Send OTP verification code
      await this.auth.api.sendVerificationOTP({
        body: {
          email,
          type: "sign-in", // Use sign-in type since user is now created
        },
      });

      return { success: true };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  /**
   * Initiates sign-in by sending an OTP code to an existing user's email.
   *
   * Requirements: 2.2, 2.6, 8.1, 8.4
   *
   * @param params - Parameters containing email and optional request metadata
   * @returns Result of the sign-in attempt
   */
  async signInViaEmail(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
    bypassRadar?: boolean;
  }): Promise<EmailAuthResult> {
    const { email, ipAddress, userAgent, bypassRadar } = params;

    // Bot detection check (unless bypassed after Turnstile verification)
    if (!bypassRadar) {
      const botDecision = this.checkBotRisk({ email, ipAddress, userAgent });

      if (botDecision === "challenge") {
        return {
          success: false,
          code: AuthErrorCode.RADAR_CHALLENGE_REQUIRED,
          message: "Please complete the verification challenge to continue.",
          email,
          challengeParams: {
            ipAddress,
            userAgent,
            authMethod: "Email_OTP",
            action: "sign-in",
          },
        };
      }

      if (botDecision === "block") {
        return this.handleError(new Error(AuthErrorCode.RADAR_BLOCKED));
      }
    }

    try {
      // Verify user exists before sending OTP using admin listUsers API
      const usersResult = await this.auth.api.listUsers({
        query: {
          searchField: "email",
          searchValue: email,
          searchOperator: "contains",
        },
      });

      if (!usersResult?.users || usersResult.users.length === 0) {
        return this.handleError(new Error(AuthErrorCode.ACCOUNT_NOT_FOUND));
      }

      // Send OTP verification code for sign-in
      await this.auth.api.sendVerificationOTP({
        body: {
          email,
          type: "sign-in",
        },
      });

      return { success: true };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  /**
   * Verifies an OTP code and authenticates the user.
   *
   * Requirements: 2.3, 2.4
   *
   * @param params - Parameters containing the email, verification code, and optional invitation token
   * @returns Result of the verification process, including redirect information on success
   */
  async verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
  }): Promise<VerificationResult> {
    const { email, code, invitationToken } = params;

    try {
      // Verify OTP and sign in
      const result = await this.auth.api.signInEmailOTP({
        body: {
          email,
          otp: code,
        },
      });

      if (!result?.token) {
        throw new Error("No session token returned");
      }

      const sessionToken = result.token;

      // If there's an invitation token, accept the invitation
      if (invitationToken) {
        try {
          await this.auth.api.acceptInvitation({
            body: {
              invitationId: invitationToken,
            },
            headers: buildSessionHeaders(sessionToken),
          });
        } catch (_inviteError) {
          // Log but don't fail - user is authenticated, invitation acceptance is secondary
          console.warn(`[Better Auth] Failed to accept invitation: ${invitationToken}`);
        }
      }

      // Check if user belongs to any organizations
      const user = result.user;
      const orgsResult = await this.auth.api.listOrganizations({
        headers: buildSessionHeaders(sessionToken),
      });

      const orgs = Array.isArray(orgsResult) ? orgsResult : [];

      // No orgs - redirect to workspace creation
      if (orgs.length === 0) {
        return {
          success: true,
          redirectTo: "/new",
          cookies: [
            {
              name: BETTER_AUTH_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions() },
            },
          ],
        };
      }

      // Multiple orgs - require selection
      if (orgs.length > 1) {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: "Please choose a workspace to continue authentication.",
          user: user ? transformUser(user as BetterAuthUser) : undefined,
          organizations: orgs.map((org) => transformOrganization(org as BetterAuthOrganization)),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions(), maxAge: 60 * 10 }, // 10 minutes
            },
          ],
        };
      }

      // Single org - set as active
      const setActiveResult = await this.auth.api.setActiveOrganization({
        body: {
          organizationId: orgs[0].id,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      // Get the updated session with activeOrganizationId set
      // IMPORTANT: disableCookieCache forces a fresh DB read after setActiveOrganization
      const updatedSession = await this.auth.api.getSession({
        query: {
          disableCookieCache: true,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!updatedSession?.session?.token) {
        throw new Error("No session returned after setting active organization");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: BETTER_AUTH_SESSION_COOKIE,
            value: updatedSession.session.token,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  /**
   * Verifies a user's email address using a verification code.
   *
   * Requirements: 10.2, 10.3
   *
   * @param params - Parameters containing the verification code and token
   * @returns Result of the email verification process
   */
  async verifyEmail(params: {
    code: string;
    token: string;
  }): Promise<VerificationResult> {
    const { code, token } = params;

    try {
      // Verify email with the OTP code - Better Auth verifyEmail expects email in body
      // We need to get the email from the session first
      const currentSession = await this.auth.api.getSession({
        headers: buildSessionHeaders(token),
      });

      if (!currentSession?.user?.email) {
        throw new Error("No user email found in session");
      }

      await this.auth.api.verifyEmailOTP({
        body: {
          email: currentSession.user.email,
          otp: code,
        },
        headers: buildSessionHeaders(token),
      });

      // Get session after email verification
      const session = await this.auth.api.getSession({
        headers: buildSessionHeaders(token),
      });

      if (!session?.session?.token) {
        throw new Error("No session returned after email verification");
      }

      const sessionToken = session.session.token;

      // Check for org selection requirement
      const orgsResult = await this.auth.api.listOrganizations({
        headers: buildSessionHeaders(sessionToken),
      });

      const orgs = Array.isArray(orgsResult) ? orgsResult : [];

      // No orgs - redirect to workspace creation
      if (orgs.length === 0) {
        return {
          success: true,
          redirectTo: "/new",
          cookies: [
            {
              name: BETTER_AUTH_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions() },
            },
          ],
        };
      }

      // Multiple orgs - require selection
      if (orgs.length > 1) {
        const user = session.user;
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: "Please choose a workspace to continue authentication.",
          user: transformUser(user as BetterAuthUser),
          organizations: orgs.map((org) => transformOrganization(org as BetterAuthOrganization)),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions(), maxAge: 60 * 10 },
            },
          ],
        };
      }

      // Single org - set as active
      await this.auth.api.setActiveOrganization({
        body: {
          organizationId: orgs[0].id,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      // Get the updated session with activeOrganizationId set
      // IMPORTANT: disableCookieCache forces a fresh DB read after setActiveOrganization
      const updatedEmailSession = await this.auth.api.getSession({
        query: {
          disableCookieCache: true,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!updatedEmailSession?.session?.token) {
        throw new Error("No session returned after setting active organization");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: BETTER_AUTH_SESSION_COOKIE,
            value: updatedEmailSession.session.token,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  /**
   * Resends an OTP code to the specified email address.
   *
   * Requirements: 2.5
   *
   * @param email - The email address to resend the auth code to
   * @returns Result of the resend attempt
   */
  async resendAuthCode(email: string): Promise<EmailAuthResult> {
    try {
      await this.auth.api.sendVerificationOTP({
        body: {
          email,
          type: "sign-in",
        },
      });

      return { success: true };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  // ============================================================================
  // Session Management Methods (Task 5.1)
  // ============================================================================

  /**
   * Validates a session token and returns information about its validity.
   * Looks up the session by token and transforms to SessionValidationResult
   * with userId, orgId, role, and impersonator information.
   *
   * Requirements: 4.2, 4.3, 4.4, 4.5, 9.2, 9.3
   *
   * @param sessionToken - The session token to validate
   * @returns Information about the session including validity, user ID, and organization
   */
  async validateSession(sessionToken: string): Promise<SessionValidationResult> {
    try {
      // Look up session by token using Better Auth API
      // Better Auth expects the session token in a cookie header, not as Bearer token
      const sessionResult = await this.auth.api.getSession({
        headers: {
          cookie: `better-auth.session_token=${sessionToken}`,
        },
      });

      // If no session found, return invalid result
      if (!sessionResult?.session || !sessionResult?.user) {
        return {
          isValid: false,
          shouldRefresh: false,
        };
      }

      const { session, user } = sessionResult;

      // Get the user's role in the active organization
      let role: string | null = null;
      if (session.activeOrganizationId) {
        try {
          const membersResult = await this.auth.api.listMembers({
            query: {
              organizationId: session.activeOrganizationId,
            },
            headers: {
              cookie: `better-auth.session_token=${sessionToken}`,
            },
          });

          const members = Array.isArray(membersResult?.members) ? membersResult.members : [];
          const activeMember = members.find(
            (m: { userId: string; role: string }) => m.userId === user.id,
          );
          role = activeMember?.role ?? null;
        } catch {
          // If we can't get members, continue without role
          role = null;
        }
      }

      // Handle impersonation metadata
      let impersonator: { email: string; reason?: string | null } | undefined;
      const impersonatedBy = (session as { impersonatedBy?: string }).impersonatedBy;
      if (impersonatedBy) {
        try {
          // Look up the impersonator user to get their email using filterField
          const impersonatorResult = await this.auth.api.listUsers({
            query: {
              filterField: "id",
              filterValue: impersonatedBy,
              filterOperator: "eq",
              limit: 1,
            },
          });

          const impersonatorUser = impersonatorResult?.users?.[0];
          if (impersonatorUser?.email) {
            impersonator = {
              email: impersonatorUser.email,
              reason: null, // Better Auth doesn't store impersonation reason
            };
          }
        } catch {
          // If we can't look up impersonator, continue without it
        }
      }

      return {
        isValid: true,
        shouldRefresh: false, // Better Auth auto-refreshes sessions
        token: session.token,
        userId: user.id,
        orgId: session.activeOrganizationId ?? null,
        role,
        impersonator,
      };
    } catch {
      // Any error means invalid session
      return {
        isValid: false,
        shouldRefresh: false,
      };
    }
  }

  /**
   * Refreshes an existing session token and returns a new token.
   * Better Auth auto-refreshes sessions, so we re-validate and return the session.
   *
   * Requirements: 4.3, 9.3
   *
   * @param sessionToken - The session token to refresh
   * @returns A new session token and related session information
   */
  async refreshSession(sessionToken: string): Promise<SessionRefreshResult> {
    try {
      // Better Auth auto-refreshes sessions, so we just get the current session
      // Better Auth expects the session token in a cookie header, not as Bearer token
      const sessionResult = await this.auth.api.getSession({
        headers: {
          cookie: `better-auth.session_token=${sessionToken}`,
        },
      });

      if (!sessionResult?.session || !sessionResult?.user) {
        throw new Error("Session not found or expired");
      }

      const { session, user } = sessionResult;

      // Get the user's role in the active organization
      let role: string | null = null;
      if (session.activeOrganizationId) {
        try {
          const membersResult = await this.auth.api.listMembers({
            query: {
              organizationId: session.activeOrganizationId,
            },
            headers: {
              cookie: `better-auth.session_token=${sessionToken}`,
            },
          });

          const members = Array.isArray(membersResult?.members) ? membersResult.members : [];
          const activeMember = members.find(
            (m: { userId: string; role: string }) => m.userId === user.id,
          );
          role = activeMember?.role ?? null;
        } catch {
          // If we can't get members, continue without role
          role = null;
        }
      }

      // Handle impersonation metadata
      let impersonator: { email: string; reason?: string | null } | undefined;
      const impersonatedBy = (session as { impersonatedBy?: string }).impersonatedBy;
      if (impersonatedBy) {
        try {
          // Look up the impersonator user to get their email using filterField
          const impersonatorResult = await this.auth.api.listUsers({
            query: {
              filterField: "id",
              filterValue: impersonatedBy,
              filterOperator: "eq",
              limit: 1,
            },
          });

          const impersonatorUser = impersonatorResult?.users?.[0];
          if (impersonatorUser?.email) {
            impersonator = {
              email: impersonatorUser.email,
              reason: null,
            };
          }
        } catch {
          // If we can't look up impersonator, continue without it
        }
      }

      // Calculate expiry - Better Auth sessions have expiresAt
      const expiresAt = session.expiresAt
        ? new Date(session.expiresAt)
        : new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);

      return {
        newToken: session.token,
        expiresAt,
        session: {
          userId: user.id,
          orgId: session.activeOrganizationId ?? null,
          role,
        },
        impersonator,
      };
    } catch (error: unknown) {
      throw error instanceof Error ? error : new Error("Failed to refresh session");
    }
  }

  /**
   * Gets the URL to redirect users to for signing out.
   * Better Auth uses server-side sign-out, so no URL is needed.
   *
   * @returns null (Better Auth uses server-side sign-out)
   */
  async getSignOutUrl(): Promise<string | null> {
    // Better Auth uses server-side sign-out, no URL needed
    return null;
  }

  // ============================================================================
  // OAuth Methods (Task 5.2)
  // ============================================================================

  /**
   * Initiates OAuth-based authentication with the specified provider.
   *
   * Note: When using Better Auth, OAuth is handled client-side using the
   * Better Auth client's signIn.social() method. This method is kept for
   * interface compatibility but should not be called directly.
   *
   * Requirements: 3.1, 3.2
   *
   * @param options - OAuth configuration including provider type and redirect URL
   * @returns Empty string - OAuth is handled client-side for Better Auth
   */
  signInViaOAuth(_options: SignInViaOAuthOptions): string {
    // OAuth is handled client-side using Better Auth client's signIn.social()
    // This method exists for interface compatibility with BaseAuthProvider
    // The actual OAuth flow is initiated from the frontend using:
    // betterAuthClient.signIn.social({ provider, callbackURL, disableRedirect: true })
    return "";
  }

  /**
   * Completes the OAuth sign-in process after the user is redirected back.
   * Better Auth's catch-all route handles the OAuth callback and sets session cookies.
   * This method extracts the session from the request and handles org selection / email verification.
   *
   * Requirements: 3.3, 3.4, 3.5, 3.6, 10.2
   *
   * @param callbackRequest - The request object from the OAuth provider callback
   * @returns Result of the OAuth authentication process
   */
  async completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult> {
    try {
      const url = new URL(callbackRequest.url);
      const error = url.searchParams.get("error");

      // Check for OAuth error from provider
      if (error) {
        return this.handleError(new Error(AuthErrorCode.UNKNOWN_ERROR));
      }

      // Better Auth handles the OAuth callback through its catch-all route
      // After the callback, the session is stored in cookies
      // We need to get the session from the request headers (cookies)
      const sessionResult = await this.auth.api.getSession({
        headers: callbackRequest.headers,
      });

      if (!sessionResult?.session || !sessionResult?.user) {
        // No session found - check if there's a code to exchange
        const code = url.searchParams.get("code");
        if (!code) {
          return this.handleError(new Error(AuthErrorCode.MISSING_REQUIRED_FIELDS));
        }
        // If we have a code but no session, the callback hasn't been processed yet
        // This shouldn't happen in normal flow as Better Auth handles the callback
        return this.handleError(new Error(AuthErrorCode.UNKNOWN_ERROR));
      }

      const { session, user } = sessionResult;
      const sessionToken = session.token;

      // Check if email verification is required
      if (!user.emailVerified) {
        return {
          success: false,
          code: AuthErrorCode.EMAIL_VERIFICATION_REQUIRED,
          message: "Email address not verified. Please check your email for a verification code.",
          user: transformUser(user as unknown as BetterAuthUser),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions(), maxAge: 60 * 10 }, // 10 minutes
            },
          ],
        };
      }

      // Check if user belongs to any organizations
      const orgsResult = await this.auth.api.listOrganizations({
        headers: buildSessionHeaders(sessionToken),
      });

      const orgs = Array.isArray(orgsResult) ? orgsResult : [];

      // No orgs - redirect to workspace creation
      if (orgs.length === 0) {
        return {
          success: true,
          redirectTo: "/new",
          cookies: [
            {
              name: BETTER_AUTH_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions() },
            },
          ],
        };
      }

      // Multiple orgs - require selection
      if (orgs.length > 1) {
        return {
          success: false,
          code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED,
          message: "Please choose a workspace to continue authentication.",
          user: transformUser(user as unknown as BetterAuthUser),
          organizations: orgs.map((org) => transformOrganization(org as BetterAuthOrganization)),
          cookies: [
            {
              name: PENDING_SESSION_COOKIE,
              value: sessionToken,
              options: { ...getAuthCookieOptions(), maxAge: 60 * 10 }, // 10 minutes
            },
          ],
        };
      }

      // Single org - set as active
      await this.auth.api.setActiveOrganization({
        body: {
          organizationId: orgs[0].id,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      // Get the updated session with activeOrganizationId set
      // IMPORTANT: disableCookieCache forces a fresh DB read after setActiveOrganization
      const updatedOAuthSession = await this.auth.api.getSession({
        query: {
          disableCookieCache: true,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!updatedOAuthSession?.session?.token) {
        throw new Error("No session returned after setting active organization");
      }

      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: BETTER_AUTH_SESSION_COOKIE,
            value: updatedOAuthSession.session.token,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  /**
   * Completes organization selection during multi-step authentication.
   * Sets the active organization and creates a session cookie.
   *
   * Requirements: 5.4
   *
   * @param params - Parameters containing orgId and pending auth token
   * @returns Result with session cookie on success
   */
  async completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult> {
    const { orgId, pendingAuthToken } = params;

    try {
      // Set the active organization
      await this.auth.api.setActiveOrganization({
        body: {
          organizationId: orgId,
        },
        headers: buildSessionHeaders(pendingAuthToken),
      });

      // Verify activeOrganizationId was set
      const sessionResult = await this.auth.api.getSession({
        query: {
          disableCookieCache: true,
        },
        headers: buildSessionHeaders(pendingAuthToken),
      });

      if (!sessionResult?.session || !sessionResult?.user) {
        throw new Error("No session returned after org selection");
      }

      // IMPORTANT: Use the original pendingAuthToken, NOT session.token
      // session.token is the session ID (32 chars), but we need the full token (77 chars)
      // that includes the signature for cookie authentication
      return {
        success: true,
        redirectTo: "/apis",
        cookies: [
          {
            name: BETTER_AUTH_SESSION_COOKIE,
            value: pendingAuthToken,
            options: { ...getAuthCookieOptions() },
          },
        ],
      };
    } catch (error: unknown) {
      return this.mapBetterAuthError(error);
    }
  }

  // ============================================================================
  // User Management Methods (Task 8.1)
  // ============================================================================

  /**
   * Looks up a user by ID via Better Auth admin API.
   *
   * Requirements: 6.1
   *
   * @param userId - The user ID to look up
   * @returns The User object or null if not found
   */
  async getUser(userId: string): Promise<User | null> {
    try {
      // Get the session token to fetch user data from the cookie cache
      // Cookie caching is enabled in better-auth-server.ts with 5-minute maxAge
      // This avoids DB queries by reading from the signed session cookie
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        return null;
      }

      // URL decode the token if it contains encoded characters
      const decodedToken = sessionToken.includes("%")
        ? decodeURIComponent(sessionToken)
        : sessionToken;

      // Get user data from the cached session cookie
      const sessionResult = await this.auth.api.getSession({
        headers: buildSessionHeaders(decodedToken),
      });

      if (!sessionResult?.user) {
        return null;
      }

      const user = sessionResult.user;

      // Verify the user ID matches what we're looking for
      if (user.id !== userId) {
        return null;
      }

      return transformUser(user as BetterAuthUser);
    } catch (error) {
      console.error(`[Better Auth] Error fetching user ${userId}:`, error);
      return null;
    }
  }

  /**
   * Looks up a user by email via Better Auth admin API.
   *
   * Requirements: 6.1
   *
   * @param email - The email address to look up
   * @returns The User object or null if not found
   */
  async findUser(email: string): Promise<User | null> {
    try {
      const usersResult = await this.auth.api.listUsers({
        query: {
          searchField: "email",
          searchValue: email,
          searchOperator: "contains",
          limit: 1,
        },
      });

      const user = usersResult?.users?.[0];
      if (!user) {
        return null;
      }

      return transformUser(user as BetterAuthUser);
    } catch {
      return null;
    }
  }

  // ============================================================================
  // Organization Management Methods (Task 7.1)
  // ============================================================================

  /**
   * Creates a new organization (tenant) for a specific user.
   * The user is auto-assigned as admin due to organization plugin config.
   *
   * Requirements: 5.1, 5.2
   *
   * @param params - Parameters containing org name and user ID
   * @returns The created organization's ID
   */
  async createTenant(params: { name: string; userId: string }): Promise<string> {
    const { name, userId } = params;

    try {
      // Generate slug from name (lowercase, replace spaces with hyphens)
      const slug = name
        .toLowerCase()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]/g, "");

      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      // Create organization - Better Auth's organization plugin with creatorRole: "admin"
      // auto-assigns the creator as admin member
      const orgResult = await this.auth.api.createOrganization({
        body: {
          name,
          slug,
          userId, // The user who will be the creator/admin
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!orgResult?.id) {
        throw new Error("Failed to create organization");
      }

      return orgResult.id;
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to create tenant");
    }
  }

  /**
   * Creates a new organization (internal/protected method).
   *
   * Requirements: 5.2
   *
   * @param name - The organization name
   * @returns The created Organization object
   */
  protected async createOrg(name: string): Promise<Organization> {
    try {
      // Generate slug from name
      const slug = name
        .toLowerCase()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]/g, "");

      const orgResult = await this.auth.api.createOrganization({
        body: {
          name,
          slug,
        },
      });

      if (!orgResult) {
        throw new Error("Failed to create organization");
      }

      return transformOrganization(orgResult as BetterAuthOrganization);
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to create organization");
    }
  }

  /**
   * Gets an organization by ID.
   *
   * Requirements: 5.4
   *
   * @param orgId - The organization ID
   * @returns The Organization object
   */
  async getOrg(orgId: string): Promise<Organization> {
    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      const orgResult = await this.auth.api.getFullOrganization({
        query: {
          organizationId: orgId,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!orgResult) {
        throw new Error("Organization not found");
      }

      return transformOrganization(orgResult as BetterAuthOrganization);
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to get organization");
    }
  }

  /**
   * Updates an organization's name.
   *
   * Requirements: 5.5
   *
   * @param params - Parameters containing org ID and new name
   * @returns The updated Organization object
   */
  async updateOrg(params: UpdateOrgParams): Promise<Organization> {
    const { id, name } = params;

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      const orgResult = await this.auth.api.updateOrganization({
        body: {
          organizationId: id,
          data: {
            name,
          },
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!orgResult) {
        throw new Error("Failed to update organization");
      }

      return transformOrganization(orgResult as BetterAuthOrganization);
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update organization");
    }
  }

  /**
   * Switches the active organization for the current session.
   * Returns a new session scoped to the target organization.
   *
   * Requirements: 5.3
   *
   * @param newOrgId - The target organization ID
   * @returns SessionRefreshResult with the new session
   */
  async switchOrg(newOrgId: string): Promise<SessionRefreshResult> {
    try {
      // Get current session token
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      // Better Auth expects session token in cookie header
      const cookieHeader = `better-auth.session_token=${sessionToken}`;

      // Set the active organization
      await this.auth.api.setActiveOrganization({
        body: {
          organizationId: newOrgId,
        },
        headers: {
          cookie: cookieHeader,
        },
      });

      // Get the updated session
      // IMPORTANT: disableCookieCache forces a fresh DB read after setActiveOrganization
      const sessionResult = await this.auth.api.getSession({
        query: {
          disableCookieCache: true,
        },
        headers: {
          cookie: cookieHeader,
        },
      });

      if (!sessionResult?.session || !sessionResult?.user) {
        throw new Error("Failed to get session after org switch");
      }

      const { session, user } = sessionResult;

      // Get the user's role in the new organization
      let role: string | null = null;
      try {
        const membersResult = await this.auth.api.listMembers({
          query: {
            organizationId: newOrgId,
          },
          headers: {
            cookie: cookieHeader,
          },
        });

        const members = Array.isArray(membersResult?.members) ? membersResult.members : [];
        const activeMember = members.find(
          (m: { userId: string; role: string }) => m.userId === user.id,
        );
        role = activeMember?.role ?? null;
      } catch {
        // If we can't get members, continue without role
        role = null;
      }

      // Calculate expiry
      const expiresAt = session.expiresAt
        ? new Date(session.expiresAt)
        : new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);

      return {
        newToken: session.token,
        expiresAt,
        session: {
          userId: user.id,
          orgId: newOrgId,
          role,
        },
        impersonator: undefined,
      };
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to switch organization");
    }
  }

  // ============================================================================
  // Membership Management Methods (Task 8.1)
  // ============================================================================

  /**
   * Lists all memberships for a user across all their organizations.
   * Gets all orgs for the user, then gets members for each org, then filters to the user.
   *
   * Requirements: 6.1
   *
   * @param userId - The user ID to list memberships for
   * @returns List of memberships with user, organization, and role data
   */
  async listMemberships(userId: string): Promise<MembershipListResponse> {
    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      // Get the user first
      const user = await this.getUser(userId);
      if (!user) {
        return { data: [], metadata: {} };
      }

      // Get all organizations the user belongs to
      const orgsResult = await this.auth.api.listOrganizations({
        headers: buildSessionHeaders(sessionToken),
      });

      const orgs = Array.isArray(orgsResult) ? orgsResult : [];
      const memberships: Membership[] = [];

      // For each organization, get members and filter to the user
      for (const org of orgs) {
        try {
          const membersResult = await this.auth.api.listMembers({
            query: {
              organizationId: org.id,
            },
            headers: buildSessionHeaders(sessionToken),
          });

          const members = Array.isArray(membersResult?.members) ? membersResult.members : [];
          const userMember = members.find((m: { userId: string }) => m.userId === userId);

          if (userMember) {
            const transformedOrg = transformOrganization(org as BetterAuthOrganization);
            memberships.push(
              transformMembership(userMember as BetterAuthMember, user, transformedOrg),
            );
          }
        } catch {
          // Skip orgs we can't get members for
        }
      }

      return { data: memberships, metadata: {} };
    } catch {
      return { data: [], metadata: {} };
    }
  }

  /**
   * Lists all members of an organization.
   *
   * Requirements: 6.2
   *
   * @param orgId - The organization ID to list members for
   * @returns List of memberships with user data and roles
   */
  async getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      // Get the organization
      const org = await this.getOrg(orgId);

      // Get all members of the organization
      const membersResult = await this.auth.api.listMembers({
        query: {
          organizationId: orgId,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      const members = Array.isArray(membersResult?.members) ? membersResult.members : [];
      const memberships: Membership[] = [];

      // Transform each member, looking up user data
      for (const member of members) {
        try {
          const user = await this.getUser(member.userId);
          if (user) {
            memberships.push(transformMembership(member as BetterAuthMember, user, org));
          }
        } catch {
          // Skip members we can't get user data for
        }
      }

      return { data: memberships, metadata: {} };
    } catch {
      return { data: [], metadata: {} };
    }
  }

  /**
   * Updates a membership's role.
   *
   * Requirements: 6.3
   *
   * @param params - Parameters containing membership ID and new role
   * @returns The updated Membership object
   */
  async updateMembership(params: UpdateMembershipParams): Promise<Membership> {
    const { membershipId, role } = params;

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      // Update the member's role - Better Auth returns the member directly
      const updatedMember = await this.auth.api.updateMemberRole({
        body: {
          memberId: membershipId,
          role,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!updatedMember) {
        throw new Error("Failed to update membership role");
      }

      // Get the user and organization for the membership
      const user = await this.getUser(updatedMember.userId);
      if (!user) {
        throw new Error("User not found for membership");
      }

      const org = await this.getOrg(updatedMember.organizationId);

      return transformMembership(updatedMember as BetterAuthMember, user, org);
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update membership");
    }
  }

  /**
   * Removes a membership from an organization.
   *
   * Requirements: 6.4
   *
   * @param membershipId - The membership ID to remove
   */
  async removeMembership(membershipId: string): Promise<void> {
    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("No active session found");
      }

      await this.auth.api.removeMember({
        body: {
          memberIdOrEmail: membershipId,
        },
        headers: buildSessionHeaders(sessionToken),
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to remove membership");
    }
  }

  // ============================================================================
  // Invitation Management Methods (Task 9.1)
  // ============================================================================

  /**
   * Invites a user to an organization by email.
   * Creates an invitation with a unique token, specified role, and pending state.
   *
   * Requirements: 7.1
   *
   * @param params - Parameters containing orgId, email, and role
   * @returns The created Invitation object
   */
  async inviteMember(params: OrgInviteParams): Promise<Invitation> {
    const { orgId, email, role = "basic_member" } = params;

    if (!orgId || !email) {
      throw new Error("Organization id and email are required.");
    }

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("User must be authenticated to invite members.");
      }

      // Better Auth uses "member" instead of "basic_member"
      const betterAuthRole = role === "basic_member" ? "member" : role;

      const invitationResult = await this.auth.api.createInvitation({
        body: {
          organizationId: orgId,
          email,
          role: betterAuthRole,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!invitationResult) {
        throw new Error("Failed to create invitation");
      }

      return transformInvitation(invitationResult as BetterAuthInvitation);
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to invite member");
    }
  }

  /**
   * Lists invitations for an organization.
   * Returns only invitations in pending or expired states.
   *
   * Requirements: 7.2
   *
   * @param orgId - The organization ID to list invitations for
   * @returns List of pending/expired invitations
   */
  async getInvitationList(orgId: string): Promise<InvitationListResponse> {
    if (!orgId) {
      throw new Error("Organization Id is required");
    }

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        return { data: [], metadata: {} };
      }

      const invitationsResult = await this.auth.api.listInvitations({
        query: {
          organizationId: orgId,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      // listInvitations returns an array directly
      const invitations = Array.isArray(invitationsResult) ? invitationsResult : [];

      // Transform and filter to only pending or expired invitations
      const transformedInvitations = invitations
        .map((inv: BetterAuthInvitation) => transformInvitation(inv))
        .filter((inv) => inv.state === "pending" || inv.state === "expired");

      return {
        data: transformedInvitations,
        metadata: {},
      };
    } catch {
      return { data: [], metadata: {} };
    }
  }

  /**
   * Gets a single invitation by its token/ID.
   *
   * Requirements: 7.3
   *
   * @param invitationToken - The invitation token/ID to look up
   * @returns The Invitation object or null if not found
   */
  async getInvitation(invitationToken: string): Promise<Invitation | null> {
    if (!invitationToken) {
      return null;
    }

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        return null;
      }

      const invitationResult = await this.auth.api.getInvitation({
        query: {
          id: invitationToken,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!invitationResult) {
        return null;
      }

      return transformInvitation(invitationResult as BetterAuthInvitation);
    } catch {
      return null;
    }
  }

  /**
   * Revokes/cancels an invitation.
   * Updates the invitation state to revoked.
   *
   * Requirements: 7.4
   *
   * @param invitationId - The invitation ID to revoke
   */
  async revokeOrgInvitation(invitationId: string): Promise<void> {
    if (!invitationId) {
      throw new Error("Invitation Id is required");
    }

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("User must be authenticated to revoke invitations.");
      }

      await this.auth.api.cancelInvitation({
        body: {
          invitationId,
        },
        headers: buildSessionHeaders(sessionToken),
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to revoke invitation");
    }
  }

  /**
   * Accepts an invitation and adds the user to the organization.
   * Updates the invitation state to accepted.
   *
   * Requirements: 7.3, 7.5
   *
   * @param invitationId - The invitation ID to accept
   * @returns The updated Invitation object
   */
  async acceptInvitation(invitationId: string): Promise<Invitation> {
    if (!invitationId) {
      throw new Error("Invitation Id is required");
    }

    try {
      // Get current session token for authenticated request
      const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);
      if (!sessionToken) {
        throw new Error("User must be authenticated to accept invitations.");
      }

      const result = await this.auth.api.acceptInvitation({
        body: {
          invitationId,
        },
        headers: buildSessionHeaders(sessionToken),
      });

      if (!result?.invitation) {
        throw new Error("Failed to accept invitation");
      }

      return transformInvitation(result.invitation as BetterAuthInvitation);
    } catch (error: unknown) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to accept invitation");
    }
  }
}
