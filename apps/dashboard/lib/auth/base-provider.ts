

import { NextRequest, NextResponse } from 'next/server';
import {
  User,
  Organization,
  Membership,
  SessionValidationResult,
  SessionData,
  MembershipListResponse,
  InvitationListResponse,
  UpdateOrgParams,
  UpdateMembershipParams,
  OrgInviteParams,
  UserData,
  SignInViaOAuthOptions,
  Invitation,
  AuthErrorResponse,
  AuthErrorCode,
  errorMessages,
  EmailAuthResult,
  VerificationResult,
  OAuthResult,
  StateChangeResponse,
  NavigationResponse,
  MiddlewareConfig,
  DEFAULT_MIDDLEWARE_CONFIG
} from './types';
import { getCookie } from './cookies';

export abstract class BaseAuthProvider {
  // Session Management
  abstract validateSession(sessionToken: string): Promise<SessionValidationResult>;
  abstract refreshSession(orgId?: string): Promise<SessionData | null>;
  
  // Authentication
  abstract signInViaEmail(email: string): Promise<EmailAuthResult>;
  abstract verifyAuthCode(params: { email: string; code: string }): Promise<VerificationResult>;
  abstract resendAuthCode(email: string): Promise<EmailAuthResult>;
  abstract signUpViaEmail(params: UserData): Promise<EmailAuthResult>;
  abstract getSignOutUrl(): Promise<string | null>;
  abstract completeOrgSelection(params: { orgId: string; pendingAuthToken: string }): Promise<VerificationResult>;
  
  // OAuth Authentication
  abstract signInViaOAuth(options: SignInViaOAuthOptions): string;
  abstract completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;

  // User Management
  abstract getCurrentUser(): Promise<User | null>;
  abstract getUser(userId: string): Promise<User | null>;

  // Organization Management
  abstract createTenant(params: { name: string; userId: string }): Promise<string>;
  abstract updateOrg(params: UpdateOrgParams): Promise<Organization>;
  protected abstract createOrg(name: string): Promise<Organization>;
  abstract getOrg(orgId: string): Promise<Organization>;
  
  // Membership Management
  abstract listMemberships(): Promise<MembershipListResponse>;
  abstract getOrganizationMemberList(orgId: string): Promise<MembershipListResponse>;
  abstract updateMembership(params: UpdateMembershipParams): Promise<Membership>;
  abstract removeMembership(membershipId: string): Promise<void>;

  // Invitation Management
  abstract inviteMember(params: OrgInviteParams): Promise<Invitation>;
  abstract getInvitationList(orgId: string): Promise<InvitationListResponse>;
  abstract revokeOrgInvitation(invitationId: string): Promise<void>;

  // Error Handling
  protected handleError(error: unknown): AuthErrorResponse {
    console.error('Auth error:', error);
    
    if (error instanceof Error) {
      // Handle provider-specific errors
      if ('code' in error && typeof error.code === 'string') {
        const errorCode = error.code as AuthErrorCode;
        if (errorCode in AuthErrorCode) {
          return {
            success: false,
            code: errorCode,
            message: errorMessages[errorCode]
          };
        }
      }

      // Handle generic errors
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: error.message
      };
    }

    // Fallback error
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR]
    };
  }

  // Utility Methods
  protected createStateChangeResponse(): StateChangeResponse {
    return { success: true };
  }

  protected createNavigationResponse(redirectTo: string, cookies: NavigationResponse['cookies']): NavigationResponse {
    return {
      success: true,
      redirectTo,
      cookies
    };
  }

protected isPublicPath(pathname: string, publicPaths: string[]): boolean {
  const isPublic = publicPaths.some(path => pathname.startsWith(path));
  console.debug('Checking public path:', { pathname, publicPaths, isPublic });
  return isPublic;
}

protected redirectToLogin(request: NextRequest, config: MiddlewareConfig): NextResponse {
  const signInUrl = new URL(config.loginPath, request.url);
  signInUrl.searchParams.set('redirect', request.nextUrl.pathname);
  const response = NextResponse.redirect(signInUrl);
  response.cookies.delete(config.cookieName);
  return response;
}

  // Public middleware factory method
  public createMiddleware(config: Partial<MiddlewareConfig> = {}) {
    const middlewareConfig = {
      ...DEFAULT_MIDDLEWARE_CONFIG,
      ...config
    };

    return async (request: NextRequest): Promise<NextResponse> => {
      if (!middlewareConfig.enabled) {
        return NextResponse.next();
      }

      const { pathname } = request.nextUrl;
      console.debug('Middleware processing:', {
        url: request.url,
        pathname,
        publicPaths: middlewareConfig.publicPaths
      });

      if (this.isPublicPath(pathname, middlewareConfig.publicPaths)) {
        console.debug('Public path detected, proceeding without auth check');
        return NextResponse.next();
      }

      try {
        const token = await getCookie(middlewareConfig.cookieName, request);
        if (!token) {
          console.debug('No session token found, redirecting to login');
          return this.redirectToLogin(request, middlewareConfig);
        }

        const validationResult = await this.validateSession(token);
        
        if (validationResult.isValid) {
          return NextResponse.next();
        }
        
        if (validationResult.shouldRefresh) {
          try {
            await this.refreshSession();
            return NextResponse.next();
          } catch (error) {
            console.debug('Session refresh failed, redirecting to login');
            const response = this.redirectToLogin(request, middlewareConfig);
            response.cookies.delete(middlewareConfig.cookieName);
            return response;
          }
        }
        
        console.debug('Invalid session, redirecting to login');
        const response = this.redirectToLogin(request, middlewareConfig);
        response.cookies.delete(middlewareConfig.cookieName);
        return response;

      } catch (error) {
        console.error('Authentication middleware error:', {
          error: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined,
          url: request.url,
          pathname
        });
        return this.redirectToLogin(request, middlewareConfig);
      }
    };
  }
}