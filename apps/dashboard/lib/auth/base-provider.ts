import { NextRequest, NextResponse } from "next/server";
import { getCookie } from "./cookies";
import {
  type AuthProvider,
  type AuthSession,
  type User,
  type OrgMembership,
  type SignInViaOAuthOptions,
  type OAuthResult,
  type MiddlewareConfig,
  DEFAULT_MIDDLEWARE_CONFIG,
  UNKEY_SESSION_COOKIE
} from "./types";

export abstract class BaseAuthProvider implements AuthProvider {
  constructor(protected config: any = {}) {}
  [key: string]: any;

  // Public abstract methods that must be implemented
  abstract validateSession(token: string): Promise<AuthSession | null>;
  abstract getCurrentUser(): Promise<User | null>;
  abstract listMemberships(userId?: string): Promise<OrgMembership>;
  abstract signUpViaEmail(email: string): Promise<any>;
  abstract signInViaOAuth(options: SignInViaOAuthOptions): string;
  abstract completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;
  abstract signIn(orgId?: string): Promise<any>;
  abstract getSignOutUrl(): Promise<any>;
  abstract updateTenant(org: Partial<any>): Promise<any>;

  // Private utility methods
  private isPublicPath(pathname: string, publicPaths: string[]): boolean {
    const isPublic = publicPaths.some(path => pathname.startsWith(path));
    console.debug('Checking public path:', { pathname, publicPaths, isPublic });
    return isPublic;
  }

  private redirectToLogin(request: NextRequest, config: MiddlewareConfig): NextResponse {
    const signInUrl = new URL(config.loginPath, request.url);
    signInUrl.searchParams.set('redirect', request.nextUrl.pathname);
    const response = NextResponse.redirect(signInUrl);
    response.cookies.delete(config.cookieName);
    return response;
  }

  // Public middleware method
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
        const token = await getCookie(UNKEY_SESSION_COOKIE, request);
        if (!token) {
          console.debug('No session token found, redirecting to login');
          return this.redirectToLogin(request, middlewareConfig);
        }
        const session = await this.validateSession(token);
        if (!session) {
          console.debug('No validated session found, redirecting to login');
          return this.redirectToLogin(request, middlewareConfig);
        }

        console.debug('Valid session found, proceeding');
        return NextResponse.next();
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