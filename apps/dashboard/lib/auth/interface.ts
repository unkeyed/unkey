import { User } from "@workos-inc/node";
import { NextRequest, NextResponse } from "next/server";

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

  signOut(): Promise<T>;

  // update name, domain or picture
  updateOrg(org: Partial<T>): Promise<T>;
}

// Base middleware implementation that providers can extend
export abstract class BaseAuthProvider implements AuthProvider {
  constructor(protected config: any = {}) {}

  abstract getSession(token: string): Promise<AuthSession | null>;

  abstract getOrgId(): Promise<any>;

  // called in RSC, giving us some display data for the user
  abstract getUser(): Promise<any | null>;

  abstract listOrganizations(): Promise<any>;

  abstract signUpViaEmail(email: string): Promise<any>;

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
        return await this.handleMiddlewareRequest(request, middlewareConfig);
      } catch (error) {
        console.error('Authentication middleware error:', error);
        return this.redirectToLogin(request, middlewareConfig);
      }
    };
  }

  protected async handleMiddlewareRequest(
    request: NextRequest,
    config: MiddlewareConfig
  ): Promise<NextResponse> {
    const { pathname } = request.nextUrl;

    if (this.isPublicPath(pathname, config.publicPaths)) {
      return NextResponse.next();
    }

    const session = await this.validateSession(request, config);
    if (!session) {
      return this.redirectToLogin(request, config);
    }

    return NextResponse.next();
  }

  protected isPublicPath(pathname: string, publicPaths: string[]): boolean {
    return publicPaths.some(path => pathname.startsWith(path));
  }

  protected async validateSession(request: NextRequest, config: MiddlewareConfig) {
    const token = request.cookies.get(config.cookieName)?.value;
    if (!token) return null;

    return this.getSession(token);
  }

  protected redirectToLogin(request: NextRequest, config: MiddlewareConfig): NextResponse {
    const signInUrl = new URL(config.loginPath, request.url);
    signInUrl.searchParams.set('redirect', request.nextUrl.pathname);
    
    const response = NextResponse.redirect(signInUrl);
    response.cookies.delete(config.cookieName);
    
    return response;
  }

  public static getMatcher() {
    return ['/((?!_next/static|_next/image|favicon.ico|public/).*)'];
  }
}
