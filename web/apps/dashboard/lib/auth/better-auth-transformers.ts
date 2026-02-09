import type { Invitation, Membership, Organization, User } from "./types";

/**
 * Better Auth type definitions for transformation.
 * These represent the shape of data returned by Better Auth APIs.
 */
export interface BetterAuthUser {
  id: string;
  email: string;
  name: string | null;
  image: string | null;
  emailVerified: boolean;
  createdAt: Date;
  updatedAt: Date;
}

export interface BetterAuthOrganization {
  id: string;
  name: string;
  slug: string;
  logo: string | null;
  metadata: string | null;
  createdAt: Date;
}

export interface BetterAuthMember {
  id: string;
  userId: string;
  organizationId: string;
  role: string;
  createdAt: Date;
}

export interface BetterAuthInvitation {
  id: string;
  email: string;
  inviterId: string;
  organizationId: string;
  role: string;
  status: "pending" | "accepted" | "rejected" | "canceled";
  expiresAt: Date;
  createdAt: Date;
}

/**
 * Extracts the first name from a full name string.
 * Returns the first whitespace-separated token.
 *
 * @param name - Full name string or null
 * @returns First name or null if input is null/empty
 */
export function extractFirstName(name: string | null): string | null {
  if (!name) {
    return null;
  }
  const parts = name.trim().split(/\s+/);
  return parts[0] ?? null;
}

/**
 * Extracts the last name from a full name string.
 * Returns all tokens after the first, joined by spaces.
 *
 * @param name - Full name string or null
 * @returns Last name or null if input is null/empty or single word
 */
export function extractLastName(name: string | null): string | null {
  if (!name) {
    return null;
  }
  const parts = name.trim().split(/\s+/);
  return parts.length > 1 ? parts.slice(1).join(" ") : null;
}

/**
 * Transforms a Better Auth user to the Unkey User type.
 * Splits the single `name` field into firstName/lastName.
 */
export function transformUser(baUser: BetterAuthUser): User {
  return {
    id: baUser.id,
    email: baUser.email,
    firstName: extractFirstName(baUser.name),
    lastName: extractLastName(baUser.name),
    avatarUrl: baUser.image,
    fullName: baUser.name,
  };
}

/**
 * Transforms a Better Auth organization to the Unkey Organization type.
 * Note: Better Auth org doesn't have updatedAt, so we use createdAt for both.
 */
export function transformOrganization(baOrg: BetterAuthOrganization): Organization {
  const createdAtStr = baOrg.createdAt.toISOString();
  return {
    id: baOrg.id,
    name: baOrg.name,
    createdAt: createdAtStr,
    updatedAt: createdAtStr,
  };
}

/**
 * Transforms a Better Auth member to the Unkey Membership type.
 * Requires the associated user and organization to be provided.
 */
export function transformMembership(
  baMember: BetterAuthMember,
  user: User,
  org: Organization,
): Membership {
  const createdAtStr = baMember.createdAt.toISOString();
  return {
    id: baMember.id,
    user,
    organization: org,
    role: baMember.role,
    createdAt: createdAtStr,
    updatedAt: createdAtStr,
    status: "active",
  };
}

/**
 * Maps Better Auth invitation status to Unkey invitation state.
 * Better Auth uses "canceled" while Unkey uses "revoked".
 */
function mapInvitationStatus(
  status: BetterAuthInvitation["status"],
  expiresAt: Date,
): Invitation["state"] {
  if (status === "canceled") {
    return "revoked";
  }
  if (status === "accepted") {
    return "accepted";
  }
  // Check if pending invitation has expired
  if (status === "pending" && expiresAt < new Date()) {
    return "expired";
  }
  return "pending";
}

/**
 * Transforms a Better Auth invitation to the Unkey Invitation type.
 * Maps status values and generates a token from the invitation ID.
 */
export function transformInvitation(baInvitation: BetterAuthInvitation): Invitation {
  const state = mapInvitationStatus(baInvitation.status, baInvitation.expiresAt);
  const createdAtStr = baInvitation.createdAt.toISOString();

  return {
    id: baInvitation.id,
    email: baInvitation.email,
    state,
    acceptedAt: state === "accepted" ? createdAtStr : null,
    revokedAt: state === "revoked" ? createdAtStr : null,
    expiresAt: baInvitation.expiresAt.toISOString(),
    token: baInvitation.id, // Better Auth uses invitation ID as token
    organizationId: baInvitation.organizationId,
    inviterUserId: baInvitation.inviterId,
    createdAt: createdAtStr,
    updatedAt: createdAtStr,
  };
}
