import { type NextRequest, NextResponse } from "next/server";
import { getCookie } from "./cookies";
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

export abstract class BaseAuthProvider {
  // Session Management
  abstract validateSession(sessionToken: string): Promise<SessionValidationResult>;
  abstract refreshSession(sessionToken: string): Promise<SessionRefreshResult>;

  // Authentication
  abstract signInViaEmail(email: string): Promise<EmailAuthResult>;
  abstract verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
  }): Promise<VerificationResult>;
  abstract verifyEmail(params: { code: string; token: string }): Promise<VerificationResult>;
  abstract resendAuthCode(email: string): Promise<EmailAuthResult>;
  abstract signUpViaEmail(params: UserData): Promise<EmailAuthResult>;
  abstract getSignOutUrl(): Promise<string | null>;
  abstract completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult>;

  // OAuth Authentication
  abstract signInViaOAuth(options: SignInViaOAuthOptions): string;
  abstract completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;

  // User Management
  abstract getCurrentUser(): Promise<User | null>;
  abstract getUser(userId: string): Promise<User | null>;
  abstract findUser(email: string): Promise<User | null>;

  // Organization Management
  abstract createTenant(params: { name: string; userId: string }): Promise<string>;
  abstract updateOrg(params: UpdateOrgParams): Promise<Organization>;
  protected abstract createOrg(name: string): Promise<Organization>;
  abstract getOrg(orgId: string): Promise<Organization>;
  abstract switchOrg(newOrgId: string): Promise<SessionRefreshResult>;

  // Membership Management
  abstract listMemberships(userId: string): Promise<MembershipListResponse>;
  abstract getOrganizationMemberList(orgId: string): Promise<MembershipListResponse>;
  abstract updateMembership(params: UpdateMembershipParams): Promise<Membership>;
  abstract removeMembership(membershipId: string): Promise<void>;

  // Invitation Management
  abstract inviteMember(params: OrgInviteParams): Promise<Invitation>;
  abstract getInvitationList(orgId: string): Promise<InvitationListResponse>;
  abstract getInvitation(invitationToken: string): Promise<Invitation | null>;
  abstract revokeOrgInvitation(invitationId: string): Promise<void>;
  abstract acceptInvitation(invitationId: string): Promise<Invitation>;

  // Error Handling
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

  // Utility Methods
  protected createStateChangeResponse(): StateChangeResponse {
    return { success: true };
  }

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

  protected isPublicPath(pathname: string, publicPaths: string[]): boolean {
    const isPublic = publicPaths.some((path) => pathname.startsWith(path));
    return isPublic;
  }

  protected redirectToLogin(request: NextRequest, config: MiddlewareConfig): NextResponse {
    const signInUrl = new URL(config.loginPath, request.url);
    signInUrl.searchParams.set("redirect", request.nextUrl.pathname);
    const response = NextResponse.redirect(signInUrl);
    response.cookies.delete(config.cookieName);
    response.headers.set('x-middleware-processed', 'true');
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
