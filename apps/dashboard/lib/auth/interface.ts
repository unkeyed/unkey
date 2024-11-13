import { User } from "@workos-inc/node";
import { NextRequest, NextResponse } from "next/server";

export type OAuthStrategy = "google" | "github";

export interface SignInViaOAuthOptions {
    redirectUri?: string,
    provider: OAuthStrategy
}

// Core middleware configuration interface
export interface MiddlewareConfig {
  enabled: boolean;
  publicPaths: string[];
  loginPath: string;
  cookieName: string;
}

// Session interface that different providers can map to
export interface AuthSession {
  userId: string;
  orgId: string;
  expiresAt: Date;
  [key: string]: any; // Allow additional provider-specific fields
}

// Default middleware configuration
export const DEFAULT_MIDDLEWARE_CONFIG: MiddlewareConfig = {
  enabled: true,
  publicPaths: ['/auth/sign-in', '/auth/sign-up', '/favicon.ico'],
  loginPath: '/auth/sign-in',
  cookieName: 'auth_token',
};

export interface AuthProvider<T = any> {
  [key: string]: any;
  // If there is none, it must trigger a redirect to the sign in page.
  getOrgId(): Promise<T>;

  // called in trpc, it returns just enough to know who's talking to us
  getSession(token:string): Promise<{ userId: string; orgId: string } | null>;

  // called in RSC, giving us some display data for the user
  getUser(): Promise<any | null>;

  listOrganizations(): Promise<T>;

  signUpViaEmail(email: string): Promise<any>;

  // sign the user into a different workspace/organisation
  signIn(orgId?: string): Promise<T>;

  signInViaOAuth({}: SignInViaOAuthOptions): Response;

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

  abstract signInViaOAuth({}: SignInViaOAuthOptions): Response;

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

      try {
        // Add debug logging
        console.debug('Middleware processing:', {
          url: request.url,
          pathname: request.nextUrl.pathname,
          isPublicPath: this.isPublicPath(request.nextUrl.pathname, middlewareConfig.publicPaths),
          publicPaths: middlewareConfig.publicPaths
        });

        // If we're already on the login page, don't process further
        if (request.nextUrl.pathname === middlewareConfig.loginPath) {
          console.debug("Already on login page, returning next response");
          const response = NextResponse.next();
          console.debug("Created next response", response);
          return response;
        }

        return await this.handleMiddlewareRequest(request, middlewareConfig);
      } catch (error) {
        console.error('Authentication middleware error:', error);
        console.error('Middleware error details:', {
          error: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined,
          url: request.url,
          pathname: request.nextUrl.pathname
        });
        // Don't redirect if we're already on the login page
        if (request.nextUrl.pathname === middlewareConfig.loginPath) {
          return NextResponse.next();
        }
        return this.redirectToLogin(request, middlewareConfig);
      }
    };
  }

  protected async handleMiddlewareRequest(
    request: NextRequest,
    config: MiddlewareConfig
  ): Promise<NextResponse> {
    const { pathname } = request.nextUrl;

    // Add more detailed logging
    console.debug('Handle middleware request:', {
      pathname,
      publicPaths: config.publicPaths,
      isPublicPath: this.isPublicPath(pathname, config.publicPaths)
    });

    if (this.isPublicPath(pathname, config.publicPaths)) {
      console.debug('Public path detected, proceeding');
      return NextResponse.next();
    }

    console.debug("Validating session");
    const session = await this.validateSession(request, config);
    if (!session) {
      console.debug("No session found, redirecting to login");
      return this.redirectToLogin(request, config);
    }

    return NextResponse.next();
  }

  protected isPublicPath(pathname: string, publicPaths: string[]): boolean {
    const isPublic = publicPaths.some(path => pathname.startsWith(path));
    console.debug('Checking public path:', { pathname, publicPaths, isPublic });
    return isPublic;
  }
}
