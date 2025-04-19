import { type NextRequest, NextResponse } from "next/server";
import {
  AuthErrorCode,
  type AuthErrorResponse,
  DEFAULT_MIDDLEWARE_CONFIG,
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  type Membership,
  type MembershipListResponse,
  type MiddlewareConfig,
  type NavigationResponse,
  type OAuthResult,
  type OrgInviteParams,
  type Organization,
  type SessionRefreshResult,
  type SessionValidationResult,
  type SignInViaOAuthOptions,
  type StateChangeResponse,
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "./types";

/**
 * Abstract base class providing authentication and authorization functionality.
 * Implements core methods for session management, user authentication, and organization control.
 *
 * Extend this class to implement provider-specific authentication logic.
 */
export abstract class BaseAuthProvider {
  /**
   * Validates a user session token.
   *
   * @param sessionToken - The session token to validate
   * @returns Promise resolving to a session validation result
   */
  abstract validateSession(sessionToken: string): Promise<SessionValidationResult>;

  /**
   * Refreshes an expired access token using a refresh token.
   *
   * @param currentRefreshToken - The refresh token to use for obtaining a new access token
   * @returns Promise resolving to new session tokens
   */
  abstract refreshAccessToken(currentRefreshToken: string): Promise<SessionRefreshResult>;

  /**
   * Initiates email-based authentication by sending an authentication code.
   *
   * @param email - The email address to authenticate
   * @returns Promise resolving to email authentication result
   */
  abstract signInViaEmail(email: string): Promise<EmailAuthResult>;

  /**
   * Verifies an authentication code sent to a user's email.
   *
   * @param params - Object containing verification parameters
   * @param params.email - The email address that received the code
   * @param params.code - The authentication code to verify
   * @param params.invitationToken - Optional invitation token for invited users
   * @returns Promise resolving to verification result
   */
  abstract verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
  }): Promise<VerificationResult>;

  /**
   * Verifies a user's email address using a verification code.
   *
   * @param params - Object containing verification parameters
   * @param params.code - The verification code to check
   * @param params.token - The token associated with this verification
   * @returns Promise resolving to verification result
   */
  abstract verifyEmail(params: { code: string; token: string }): Promise<VerificationResult>;

  /**
   * Resends an authentication code to a user's email address.
   *
   * @param email - The email address to resend the code to
   * @returns Promise resolving to email authentication result
   */
  abstract resendAuthCode(email: string): Promise<EmailAuthResult>;

  /**
   * Registers a new user via email.
   *
   * @param params - User data required for registration
   * @returns Promise resolving to email authentication result
   */
  abstract signUpViaEmail(params: UserData): Promise<EmailAuthResult>;

  /**
   * Gets the URL used for signing out users.
   *
   * @returns Promise resolving to the sign-out URL or null if not applicable
   */
  abstract getSignOutUrl(): Promise<string | null>;

  /**
   * Completes the organization selection process after authentication.
   *
   * @param params - Object containing selection parameters
   * @param params.orgId - The ID of the selected organization
   * @param params.pendingAuthToken - The pending authentication token
   * @returns Promise resolving to verification result
   */
  abstract completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult>;

  /**
   * Generates a URL for OAuth authentication with a third-party provider.
   *
   * @param options - OAuth configuration options
   * @returns URL string for redirecting to the OAuth provider
   */
  abstract signInViaOAuth(options: SignInViaOAuthOptions): string;

  /**
   * Handles the OAuth callback after successful third-party authentication.
   *
   * @param callbackRequest - The request object from the OAuth callback
   * @returns Promise resolving to OAuth result
   */
  abstract completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;

  /**
   * Retrieves the currently authenticated user.
   *
   * @returns Promise resolving to the current user or null if not authenticated
   */
  abstract getCurrentUser(): Promise<User | null>;

  /**
   * Retrieves a user by their unique ID.
   *
   * @param userId - The ID of the user to retrieve
   * @returns Promise resolving to the user or null if not found
   */
  abstract getUser(userId: string): Promise<User | null>;

  /**
   * Finds a user by their email address.
   *
   * @param email - The email address to search for
   * @returns Promise resolving to the user or null if not found
   */
  abstract findUser(email: string): Promise<User | null>;

  /**
   * Creates a new tenant (organization) for a specific user.
   *
   * @param params - Object containing tenant creation parameters
   * @param params.name - The name of the tenant to create
   * @param params.userId - The ID of the user who will own the tenant
   * @returns Promise resolving to the ID of the created tenant
   */
  abstract createTenant(params: { name: string; userId: string }): Promise<string>;

  /**
   * Updates an organization's details.
   *
   * @param params - Organization update parameters
   * @returns Promise resolving to the updated organization
   */
  abstract updateOrg(params: UpdateOrgParams): Promise<Organization>;

  /**
   * Creates a new organization with the given name.
   * Protected method intended for internal use by subclasses.
   *
   * @param name - The name of the organization to create
   * @returns Promise resolving to the created organization
   */
  protected abstract createOrg(name: string): Promise<Organization>;

  /**
   * Retrieves an organization by its ID.
   *
   * @param orgId - The ID of the organization to retrieve
   * @returns Promise resolving to the organization
   */
  abstract getOrg(orgId: string): Promise<Organization>;

  /**
   * Switches the current session to a different organization.
   *
   * @param newOrgId - The ID of the organization to switch to
   * @returns Promise resolving to new session tokens
   */
  abstract switchOrg(newOrgId: string): Promise<SessionRefreshResult>;

  /**
   * Lists all memberships for a specific user.
   *
   * @param userId - The ID of the user whose memberships to list
   * @returns Promise resolving to the membership list response
   */
  abstract listMemberships(userId: string): Promise<MembershipListResponse>;

  /**
   * Retrieves all members of a specific organization.
   *
   * @param orgId - The ID of the organization to list members for
   * @returns Promise resolving to the membership list response
   */
  abstract getOrganizationMemberList(orgId: string): Promise<MembershipListResponse>;

  /**
   * Updates a user's membership properties.
   *
   * @param params - Membership update parameters
   * @returns Promise resolving to the updated membership
   */
  abstract updateMembership(params: UpdateMembershipParams): Promise<Membership>;

  /**
   * Removes a user from an organization.
   *
   * @param membershipId - The ID of the membership to remove
   * @returns Promise resolving when the membership is removed
   */
  abstract removeMembership(membershipId: string): Promise<void>;

  /**
   * Invites a new member to join an organization.
   *
   * @param params - Organization invitation parameters
   * @returns Promise resolving to the created invitation
   */
  abstract inviteMember(params: OrgInviteParams): Promise<Invitation>;

  /**
   * Retrieves all pending invitations for an organization.
   *
   * @param orgId - The ID of the organization to list invitations for
   * @returns Promise resolving to the invitation list response
   */
  abstract getInvitationList(orgId: string): Promise<InvitationListResponse>;

  /**
   * Retrieves an invitation by its token.
   *
   * @param invitationToken - The token identifying the invitation
   * @returns Promise resolving to the invitation or null if not found
   */
  abstract getInvitation(invitationToken: string): Promise<Invitation | null>;

  /**
   * Revokes a pending organization invitation.
   *
   * @param invitationId - The ID of the invitation to revoke
   * @returns Promise resolving when the invitation is revoked
   */
  abstract revokeOrgInvitation(invitationId: string): Promise<void>;

  /**
   * Accepts a pending organization invitation.
   *
   * @param invitationId - The ID of the invitation to accept
   * @returns Promise resolving to the accepted invitation
   */
  abstract acceptInvitation(invitationId: string): Promise<Invitation>;

  /**
   * Standardizes error handling across all authentication operations.
   * Converts various error types to a consistent AuthErrorResponse format.
   *
   * @param error - The error to handle
   * @returns Standardized authentication error response
   */
  protected handleError(error: unknown): AuthErrorResponse {
    console.error("Auth error:", error);

    if (error instanceof Error) {
      // Handle provider-specific errors
      if ("message" in error && typeof error.message === "string") {
        const errorCode = error.message as AuthErrorCode;
        if (errorCode in AuthErrorCode) {
          return {
            success: false,
            code: errorCode,
            message: errorMessages[errorCode],
          };
        }
      }

      // Handle generic errors
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: error.message,
      };
    }

    // Fallback error
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }

  /**
   * Creates a standard response for state change operations.
   *
   * @returns Standard state change response object
   */
  protected createStateChangeResponse(): StateChangeResponse {
    return { success: true };
  }

  /**
   * Creates a standard response for operations requiring navigation.
   *
   * @param redirectTo - The URL to redirect to
   * @param cookies - Cookies to include in the response
   * @returns Standard navigation response object
   */
  protected createNavigationResponse(
    redirectTo: string,
    cookies: NavigationResponse["cookies"],
  ): NavigationResponse {
    return {
      success: true,
      redirectTo,
      cookies,
    };
  }

  /**
   * Determines if a path is public (exempt from authentication).
   *
   * @param pathname - The path to check
   * @param publicPaths - Array of paths considered public
   * @returns True if the path is public, false otherwise
   */
  protected isPublicPath(pathname: string, publicPaths: string[]): boolean {
    const isPublic = publicPaths.some((path) => pathname.startsWith(path));
    return isPublic;
  }

  /**
   * Creates a response that redirects to the login page.
   *
   * @param request - The original request
   * @param config - Middleware configuration
   * @returns NextResponse configured to redirect to login
   */
  protected redirectToLogin(request: NextRequest, config: MiddlewareConfig): NextResponse {
    const signInUrl = new URL(config.loginPath, request.url);
    signInUrl.searchParams.set("redirect", request.nextUrl.pathname);
    const response = NextResponse.redirect(signInUrl);
    response.cookies.delete(config.cookieName);
    response.headers.set("x-middleware-processed", "true");
    return response;
  }

  /**
   * Creates a Next.js edge middleware function for basic authentication screening.
   *
   * This factory generates a middleware function that performs lightweight authentication
   * checks at the edge. It only verifies the presence of a session cookie and handles
   * public path exclusions, delegating full authentication validation to server components.
   *
   * @param config - Optional configuration to override default middleware settings
   * @returns A Next.js middleware function that performs basic auth screening and handles redirects
   *
   * @example
   * // Create middleware with custom public paths
   * const authMiddleware = authService.createMiddleware({
   *   publicPaths: ['/about', '/pricing', '/api/public'],
   *   loginPath: '/custom-login'
   * });
   *
   * // In middleware.ts
   * export default authMiddleware;
   */
  public createMiddleware(config: Partial<MiddlewareConfig> = {}) {
    const middlewareConfig = {
      ...DEFAULT_MIDDLEWARE_CONFIG,
      ...config,
    };

    return async (request: NextRequest): Promise<NextResponse> => {
      const { pathname } = request.nextUrl;

      // Skip public paths
      if (this.isPublicPath(pathname, middlewareConfig.publicPaths)) {
        return NextResponse.next();
      }

      // Check if cookie exists at all (lightweight check)
      const hasSessionCookie = request.cookies.has(middlewareConfig.cookieName);
      if (!hasSessionCookie) {
        return this.redirectToLogin(request, middlewareConfig);
      }

      // Allow request to proceed to server components for full auth check
      return NextResponse.next();
    };
  }
}
