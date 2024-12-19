import type { Cookie } from './cookies';

export const UNKEY_SESSION_COOKIE = "unkey-session";

export type OAuthStrategy = "google" | "github";

export interface User {
  id: string;
  orgId: string | null;
  email: string;
  firstName: string | null;
  lastName: string | null;
  avatarUrl: string | null;
  fullName: string | null;
}

export interface SignInViaOAuthOptions {
  redirectUrl?: string;
  redirectUrlComplete: string;
  provider: OAuthStrategy;
}

export interface MiddlewareConfig {
  enabled: boolean;
  publicPaths: string[];
  cookieName: string;
  loginPath: string;
}

export interface AuthSession {
  userId: string;
  orgId: string | null;
  [key: string]: any;
}

export interface BaseAuthResponse {
  success: boolean;
  redirectTo: string;
  cookies: Cookie[];
}

export interface OAuthSuccessResponse extends BaseAuthResponse {
  success: true;
}

export interface OAuthErrorResponse extends BaseAuthResponse {
  success: false;
  error: Error;
}

export type OAuthResult = OAuthSuccessResponse | OAuthErrorResponse;

export interface Organization {
  orgId: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

export interface Membership {
  id: string;
  orgName: string;
  orgId: string;
  role: string;
  createdAt: string;
  status: "pending" | "active" | "inactive";
}

export interface OrgMembership {
  data: Membership[];
  metadata: Record<string, unknown>;
}

export const DEFAULT_MIDDLEWARE_CONFIG: MiddlewareConfig = {
  enabled: true,
  publicPaths: ['/auth/sign-in', '/auth/sign-up', '/favicon.ico'],
  cookieName: UNKEY_SESSION_COOKIE,
  loginPath: '/auth/sign-in'
};

export interface AuthProvider<T = any> {
  validateSession(token: string): Promise<AuthSession | null>;
  getCurrentUser(): Promise<any | null>;
  listMemberships(userId?: string): Promise<OrgMembership>;
  signUpViaEmail(email: string): Promise<any>;
  signIn(orgId?: string): Promise<T>;
  signInViaOAuth(options: SignInViaOAuthOptions): String;
  completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;
  getSignOutUrl(): Promise<T>;
  updateTenant(org: Partial<T>): Promise<T>;
  [key: string]: any;
}