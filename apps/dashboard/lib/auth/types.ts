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
  id: string;
  name: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface Membership {
  id: string;
  user: User;
  organization: Organization;
  role: string;
  createdAt: string;
  updatedAt: string;
  status: "pending" | "active" | "inactive";
}

export interface OrgMembership {
  data: Membership[];
  metadata: Record<string, unknown>;
}

export interface UpdateOrgParams {
    id: string;
    name: string;
}

export interface OrgInvite {
    orgId: string;
    email: string;
    role: "basic_member" | "admin";
}
export interface OrgInvitation {
  data: Invitation[];
  metadata: Record<string, unknown>;
}
export interface Invitation {
    id: string,
    email: string,
    state: 'pending' | 'accepted' | 'revoked' | 'expired',
    acceptedAt?: string | null,
    revokedAt?: string | null,
    expiresAt: string,
    token: string,
    organizationId?: string,
    inviterUserId?: string,
    createdAt: string,
    updatedAt: string,
}
export interface SessionValidationResult {
  isValid: boolean;
  shouldRefresh: boolean;
  userId?: string;
  orgId?: string | null;
}

export interface SessionData {
  userId: string;
  orgId?: string | null;
}

export interface UpdateMembershipParams {
  membershipId: string;
  role: string;
}

export const DEFAULT_MIDDLEWARE_CONFIG: MiddlewareConfig = {
  enabled: true,
  publicPaths: ['/auth/sign-in', '/auth/sign-up', '/favicon.ico'],
  cookieName: UNKEY_SESSION_COOKIE,
  loginPath: '/auth/sign-in'
};

export interface AuthProvider<T = any> {
  validateSession(token: string): Promise<SessionValidationResult>;
  getCurrentUser(): Promise<any | null>;
  listMemberships(userId?: string): Promise<OrgMembership>;
  signUpViaEmail(email: string): Promise<any>;
  signIn(orgId?: string): Promise<T>;
  signInViaOAuth(options: SignInViaOAuthOptions): String;
  completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;
  getSignOutUrl(): Promise<T>;
  updateOrg({id, name}: UpdateOrgParams): Promise<Organization>;
  [key: string]: any;
}