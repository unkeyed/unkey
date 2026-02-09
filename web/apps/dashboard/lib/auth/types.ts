import type { Cookie } from "./cookies";

// Core Types
export const UNKEY_SESSION_COOKIE = "unkey-session";
export const UNKEY_LAST_ORG_COOKIE = "unkey_last_org_used";
export const PENDING_SESSION_COOKIE = "sess-temp";
export const SIGN_IN_URL = "/auth/sign-in";
export const SIGN_UP_URL = "/auth/sign-up";

// better auth types

export const BETTER_AUTH_SESSION_COOKIE = "better-auth.session_token";

// Local Auth consts
export const LOCAL_USER_ID = "user_local_admin";
export const LOCAL_ORG_ID = "org_localdefault"; // org IDs can only have one underscore
export const LOCAL_ORG_ROLE = "admin";

// WorkOS API endpoints
export const WORKOS_RADAR_API_URL = "https://api.workos.com/radar/attempts";

export interface User {
  id: string;
  email: string;
  firstName: string | null;
  lastName: string | null;
  avatarUrl: string | null;
  fullName: string | null;
}

export interface AuthenticatedUser extends User {
  role?: string | null;
  orgId?: string | null;
  impersonator?: {
    email: string;
    reason?: string | null;
  };
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

// Special case for Turnstile challenge
export interface PendingTurnstileResponse extends AuthErrorResponse {
  code: AuthErrorCode.RADAR_CHALLENGE_REQUIRED;
  email: string;
  challengeParams: {
    ipAddress?: string;
    userAgent?: string;
    authMethod: string;
    action: string;
  };
}

// Union types for different auth operations
export type EmailAuthResult = StateChangeResponse | AuthErrorResponse | PendingTurnstileResponse;
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
export interface SessionValidationResult {
  isValid: boolean;
  shouldRefresh: boolean;
  token?: string;
  userId?: string;
  orgId?: string | null;
  role?: string | null;
  impersonator?: {
    email: string;
    reason?: string | null;
  };
}

export interface SessionRefreshResult {
  newToken: string;
  expiresAt: Date;
  session: SessionData | null;
  impersonator?: {
    email: string;
    reason?: string | null;
  };
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
  RATE_ERROR = "RATE_ERROR",
  ACCOUNT_NOT_FOUND = "ACCOUNT_NOT_FOUND",
  ORGANIZATION_SELECTION_REQUIRED = "ORGANIZATION_SELECTION_REQUIRED",
  EMAIL_VERIFICATION_REQUIRED = "EMAIL_VERIFICATION_REQUIRED",
  PENDING_SESSION_EXPIRED = "PENDING_SESSION_EXPIRED",
  RADAR_BLOCKED = "RADAR_BLOCKED",
  RADAR_CHALLENGE_REQUIRED = "RADAR_CHALLENGE_REQUIRED",
}

export const errorMessages: Record<AuthErrorCode, string> = {
  [AuthErrorCode.EMAIL_ALREADY_EXISTS]:
    "This email address is already registered. Please sign in instead.",
  [AuthErrorCode.MISSING_REQUIRED_FIELDS]: "Please fill in all required fields.",
  [AuthErrorCode.USER_CREATION_FAILED]: "Unable to create your account. Please try again later.",
  [AuthErrorCode.INVALID_EMAIL]: "Please enter a valid email address.",
  [AuthErrorCode.NETWORK_ERROR]: "Connection error. Please check your internet and try again.",
  [AuthErrorCode.UNKNOWN_ERROR]:
    "Something went wrong. Please try again later, or contact support@unkey.com",
  [AuthErrorCode.ACCOUNT_NOT_FOUND]: "Account not found. Would you like to sign up?",
  [AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED]:
    "Please choose a workspace to continue authentication.",
  [AuthErrorCode.EMAIL_VERIFICATION_REQUIRED]:
    "Email address not verified. Please check your email for a verification code.",
  [AuthErrorCode.PENDING_SESSION_EXPIRED]:
    "Pending Authentication has expired. Please sign-in again.",
  [AuthErrorCode.RATE_ERROR]: "Limited OTP attempts",
  [AuthErrorCode.RADAR_BLOCKED]:
    "Unable to complete request due to suspicious activity. Please contact support@unkey.com if you believe this is an error.",
  [AuthErrorCode.RADAR_CHALLENGE_REQUIRED]:
    "Please complete the verification challenge to continue.",
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
