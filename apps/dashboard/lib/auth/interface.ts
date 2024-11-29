import { NextRequest, NextResponse } from "next/server";

export const UNKEY_SESSION_COOKIE = "unkey-session";

export type OAuthStrategy = "google" | "github";

export interface SignInViaOAuthOptions {
    redirectUrl?: string,
    redirectUrlComplete: string,
    provider: OAuthStrategy
}

// Core middleware configuration interface
export interface MiddlewareConfig {
  enabled: boolean;
  publicPaths: string[];
  cookieName: string;
  loginPath: string;
}

// Session interface that different providers can map to
export interface AuthSession {
  userId: string;
  orgId: string;
  [key: string]: any; // Allow additional provider-specific fields
}

// Default middleware configuration
export const DEFAULT_MIDDLEWARE_CONFIG: MiddlewareConfig = {
  enabled: true,
  publicPaths: ['/auth/sign-in', '/auth/sign-up', '/favicon.ico'], // TODO: allow glob matching
  cookieName: UNKEY_SESSION_COOKIE,
  loginPath: '/auth/sign-in'
};

export interface AuthProvider<T = any> {
  [key: string]: any;
  // If there is none, it must trigger a redirect to the sign in page.
  getOrgId(): Promise<T>;

  // called in trpc, it returns just enough to know who's talking to us
  getSession(token:string): Promise<AuthSession | null>;

  // called in RSC, giving us some display data for the user
  getUser(): Promise<any | null>;

  listOrganizations(): Promise<T>;

  signUpViaEmail(email: string): Promise<any>;

  // sign the user into a different workspace/organisation
  signIn(orgId?: string): Promise<T>;

  signInViaOAuth({}: SignInViaOAuthOptions): String;

  completeOAuthSignIn(callbackRequest: Request): void;

  signOut(): Promise<T>;

  // update name, domain or picture
  updateOrg(org: Partial<T>): Promise<T>;
}

// Base middleware implementation that providers can extend
export abstract class BaseAuthProvider implements AuthProvider {
  constructor(protected config: any = {}) {}
  [key: string]: any;

  abstract getSession(token: string): Promise<AuthSession | null>;

  abstract getOrgId(): Promise<any>;

  // called in RSC, giving us some display data for the user
  abstract getUser(): Promise<any | null>;

  abstract listOrganizations(): Promise<any>;

  abstract signUpViaEmail(email: string): Promise<any>;

  abstract signInViaOAuth({}: SignInViaOAuthOptions): String;

  abstract completeOAuthSignIn(callbackRequest: Request): void;

  // sign the user into a different workspace/organisation
  abstract signIn(orgId?: string): Promise<any>;

  abstract signOut(): Promise<any>;

  // update name, domain or picture
  abstract updateOrg(org: Partial<any>): Promise<any>;

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

      // Log initial request
      console.debug('Middleware processing:', {
        url: request.url,
        pathname,
        publicPaths: middlewareConfig.publicPaths
      });

      // Check public paths first
      if (this.isPublicPath(pathname, middlewareConfig.publicPaths)) {
        console.debug('Public path detected, proceeding without auth check');
        return NextResponse.next();
      }

      try {
        // Handle protected routes
        const session = await this.validateSession(request, middlewareConfig);
        if (!session) {
          console.debug('No session found, redirecting to login');
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

  protected isPublicPath(pathname: string, publicPaths: string[]): boolean {
    const isPublic = publicPaths.some(path => pathname.startsWith(path));
    console.debug('Checking public path:', { pathname, publicPaths, isPublic });
    return isPublic;
  }

  protected async validateSession(request: NextRequest, config: MiddlewareConfig) {
    const sessionData = request.cookies.get(config.cookieName)?.value;
    if (!sessionData) return null;

    return this.getSession(sessionData);
  }

  protected redirectToLogin(request: NextRequest, config: MiddlewareConfig): NextResponse {
    const signInUrl = new URL(config.loginPath, request.url);
    signInUrl.searchParams.set('redirect', request.nextUrl.pathname);
    
    const response = NextResponse.redirect(signInUrl);
    response.cookies.delete(config.cookieName);
    
    return response;
  }
}
