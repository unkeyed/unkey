export const UNKEY_SESSION_COOKIE = "unkey-session";
export const UNKEY_LAST_ORG_COOKIE = "unkey_last_org_used";

export const LOCAL_USER_ID = "user_local_admin";
export const LOCAL_ORG_ID = "org_localdefault";
export const LOCAL_ORG_ROLE = "admin";
export const LOCAL_AUTH_PERMISSIONS = ["admin:*"] as const;

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

export interface ListResponse<T> {
  data: T[];
  metadata: Record<string, unknown>;
}

export type MembershipListResponse = ListResponse<Membership>;
export type InvitationListResponse = ListResponse<Invitation>;

export interface SessionValidationResult {
  isValid: boolean;
  permissions?: readonly string[];
  userId?: string;
  orgId?: string | null;
  role?: string | null;
  user?: User | null;
}

export interface SessionRefreshResult {
  newToken: string;
  expiresAt: Date;
}

export interface Invitation {
  id: string;
  email: string;
  state: "pending" | "accepted" | "revoked" | "expired";
  expiresAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface UpdateOrgParams {
  id: string;
  name: string;
}

export interface UpdateMembershipParams {
  membershipId: string;
  role: string;
  orgId: string;
}

export class OrganizationScopeError extends Error {
  constructor(
    public readonly resource: "membership",
    public readonly resourceId: string,
  ) {
    super(`${resource} ${resourceId} does not belong to this organization`);
    this.name = "OrganizationScopeError";
  }
}

export interface OrgInviteParams {
  orgId: string;
  email: string;
  role: "basic_member" | "admin";
  inviterUserId: string;
}
