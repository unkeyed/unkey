import type { Cookie } from "./cookies";

// consts
export const UNKEY_SESSION_COOKIE = "unkey-session";
export const UNKEY_ACCESS_TOKEN = "unkey-access-token";
export const UNKEY_REFRESH_TOKEN = "unkey-refresh-token";
export const UNKEY_USER_IDENTITY_COOKIE = "unkey-user-identity";
export const PENDING_SESSION_COOKIE = "sess-temp";
export const SIGN_IN_URL = "/auth/sign-in";
export const SIGN_UP_URL = "/auth/sign-up";

// Token expiration (in milliseconds)
export const UNKEY_ACCESS_MAX_AGE = 5 * 60 * 1000; // 5 minutes
export const UNKEY_REFRESH_MAX_AGE = 30 * 24 * 60 * 60 * 1000; // 30 days
export const UNKEY_USER_IDENTITY_MAX_AGE = 90 * 24 * 60 * 60 * 1000; // 90 days

export interface User {
  id: string;
  orgId: string | null;
  email: string;
  firstName: string | null;
  lastName: string | null;
  avatarUrl: string | null;
  fullName: string | null;
  impersonator: unknown | undefined;
}

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

// Base response type
interface AuthResponse {
  success: boolean;
}

export type AuthResult = {
  userId: string | null;
  orgId: string | null;
  role: string | null;
  accessToken?: string | null;
  expiresAt?: Date | null;
};

// State change responses (for operations that update UI state)
export interface StateChangeResponse extends AuthResponse {
  success: true;
}

// Navigation responses (for operations that redirect/set cookies)
export interface NavigationResponse extends AuthResponse {
  success: true;
  redirectTo: string;
  cookies: Cookie[];
}

// Error response
export interface AuthErrorResponse extends AuthResponse {
  success: false;
  code: AuthErrorCode;
  message: string;
  cookies?: Cookie[]; // needed for Org selection error case
}

// Special case for org selection
export interface PendingOrgSelectionResponse extends AuthErrorResponse {
  code: AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED;
  user: User;
  organizations: Organization[];
  cookies: Cookie[];
}

// Special case for email verification
export interface PendingEmailVerificationResponse extends AuthErrorResponse {
  code: AuthErrorCode.EMAIL_VERIFICATION_REQUIRED;
  user: User;
  cookies: Cookie[];
}

// Union types for different auth operations
export type EmailAuthResult = StateChangeResponse | AuthErrorResponse;
export type VerificationResult =
  | NavigationResponse
  | PendingOrgSelectionResponse
  | AuthErrorResponse;
export type OAuthResult =
  | NavigationResponse
  | PendingOrgSelectionResponse
  | PendingEmailVerificationResponse
  | AuthErrorResponse;

// List Response Types
export interface ListResponse<T> {
  data: T[];
  metadata: Record<string, unknown>;
}

export type MembershipListResponse = ListResponse<Membership>;
export type InvitationListResponse = ListResponse<Invitation>;

// Session Types
interface BaseSessionValidationResult {
  isValid: boolean;
  shouldRefresh: boolean;
}

// valid sessions
interface ValidSessionResult extends BaseSessionValidationResult {
  isValid: true;
  sessionToken: string;
  accessToken: string;
  userId: string;
  orgId?: string | null;
  role?: string | null;
}

// invalid sessions
interface InvalidSessionResult extends BaseSessionValidationResult {
  isValid: false;
}

export type SessionValidationResult = ValidSessionResult | InvalidSessionResult;

export interface SessionRefreshResult {
  sessionToken: string;
  accessToken?: string;
  refreshToken?: string;
  expiresAt: Date;
  session: SessionData | null;
}

export interface SessionData {
  userId: string;
  orgId: string | null;
  role?: string | null;
}

// OAuth Types
export type OAuthStrategy = "google" | "github";

export interface SignInViaOAuthOptions {
  redirectUrlComplete: string;
  provider: OAuthStrategy;
}

// Invitation Types
export interface Invitation {
  id: string;
  email: string;
  state: "pending" | "accepted" | "revoked" | "expired";
  acceptedAt?: string | null;
  revokedAt?: string | null;
  expiresAt: string;
  token: string;
  organizationId?: string;
  inviterUserId?: string;
  createdAt: string;
  updatedAt: string;
}

// Operation Parameters
export interface UpdateOrgParams {
  id: string;
  name: string;
}

export interface UpdateMembershipParams {
  membershipId: string;
  role: string;
}

export interface OrgInviteParams {
  orgId: string;
  email: string;
  role: "basic_member" | "admin";
}

export interface UserData {
  firstName: string;
  lastName: string;
  email: string;
}

// Error Handling
export enum AuthErrorCode {
  EMAIL_ALREADY_EXISTS = "EMAIL_ALREADY_EXISTS",
  MISSING_REQUIRED_FIELDS = "MISSING_REQUIRED_FIELDS",
  USER_CREATION_FAILED = "USER_CREATION_FAILED",
  INVALID_EMAIL = "INVALID_EMAIL",
  NETWORK_ERROR = "NETWORK_ERROR",
  UNKNOWN_ERROR = "UNKNOWN_ERROR",
  ACCOUNT_NOT_FOUND = "ACCOUNT_NOT_FOUND",
  ORGANIZATION_SELECTION_REQUIRED = "ORGANIZATION_SELECTION_REQUIRED",
  EMAIL_VERIFICATION_REQUIRED = "EMAIL_VERIFICATION_REQUIRED",
  PENDING_SESSION_EXPIRED = "PENDING_SESSION_EXPIRED",
}

export const errorMessages: Record<AuthErrorCode, string> = {
  [AuthErrorCode.EMAIL_ALREADY_EXISTS]:
    "This email address is already registered. Please sign in instead.",
  [AuthErrorCode.MISSING_REQUIRED_FIELDS]: "Please fill in all required fields.",
  [AuthErrorCode.USER_CREATION_FAILED]: "Unable to create your account. Please try again later.",
  [AuthErrorCode.INVALID_EMAIL]: "Please enter a valid email address.",
  [AuthErrorCode.NETWORK_ERROR]: "Connection error. Please check your internet and try again.",
  [AuthErrorCode.UNKNOWN_ERROR]:
    "Something went wrong. Please try again later, or contact support@unkey.dev",
  [AuthErrorCode.ACCOUNT_NOT_FOUND]: "Account not found. Would you like to sign up?",
  [AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED]:
    "Please choose a workspace to continue authentication.",
  [AuthErrorCode.EMAIL_VERIFICATION_REQUIRED]:
    "Email address not verified. Please check your email for a verification code.",
  [AuthErrorCode.PENDING_SESSION_EXPIRED]:
    "Pending Authentication has expired. Please sign-in again.",
};

export interface MiddlewareConfig {
  enabled: boolean;
  publicPaths: string[];
  cookieName: string;
  loginPath: string;
}

export const DEFAULT_MIDDLEWARE_CONFIG: MiddlewareConfig = {
  enabled: true,
  publicPaths: ["/auth/sign-in", "/auth/sign-up", "/favicon.ico"],
  cookieName: UNKEY_SESSION_COOKIE,
  loginPath: "/auth/sign-in",
};
