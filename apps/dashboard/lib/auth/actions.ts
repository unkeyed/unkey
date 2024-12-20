'use server';

/**
 * Client-friendly actions for the auth client. 
 * You should be able to import the `auth` server provider on non-client pages.
 * But if you need access to auth functions on the client, add what you need to this file
 * and import the functions you need access to into the client.
 * 
 * Use/create accompanying hooks if you need a loading state
 * i.e. useUser, useOrganization
 * 
 * NO EDGE RUNTIME UNLESS YOU LIKE CRYTPO MODULE RESOLUTION ERRORS >:(
 */
import { auth } from './server';
import { 
  type User, 
  type OrgMembership, 
  type SignInViaOAuthOptions,
  UNKEY_SESSION_COOKIE, 
  Invitation,
  OrgInvite,
  Organization,
  OrgInvitation
} from './types';
import { deleteCookie } from './cookies';
import { redirect } from 'next/navigation';
import { db } from '../db';

// Helper function to check authentication
async function requireAuth(): Promise<User> {
  const user = await auth.getCurrentUser();
  if (!user) {
    redirect('/auth/sign-in');
  }
  return user;
}

// Helper function to check organization access
async function requireOrgAccess(orgId: string, userId: string): Promise<void> {
  const memberships = await auth.listMemberships(userId);
  const hasAccess = memberships.data.some(m => m.organization.id === orgId);
  if (!hasAccess) {
    throw new Error('You do not have access to this organization');
  }
}

// Helper to check admin status
async function requireOrgAdmin(orgId: string, userId: string): Promise<void> {
  const memberships = await auth.listMemberships(userId);
  const isAdmin = memberships.data.some(
    m => m.organization.id === orgId && m.role === 'admin'
  );
  if (!isAdmin) {
    throw new Error('This action requires admin privileges');
  }
}

export async function getCurrentUser(): Promise<User | null> {
  return await auth.getCurrentUser();
}

export async function listMemberships(userId?: string): Promise<OrgMembership> {
  const user = await requireAuth();
  // Only allow users to list their own memberships unless they provide a userId
  if (userId && userId !== user.id) {
    throw new Error('Unauthorized to view other users memberships');
  }
  return await auth.listMemberships(userId || user.id);
}

export async function refreshSession(orgId: string): Promise<void> {
  const user = await requireAuth();
  await requireOrgAccess(orgId, user.id);
  return await auth.refreshSession(orgId);
}

export async function getSignOutUrl(): Promise<string | null> {
  await requireAuth(); // Ensure user is authenticated
  const url = await auth.getSignOutUrl();
  return url;
}

export async function createTenant(params: { name: string, userId: string }): Promise<string> {
  const user = await requireAuth();
  // Only allow users to create tenants for themselves
  if (params.userId !== user.id) {
    throw new Error('Unauthorized to create tenant for another user');
  }
  return await auth.createTenant(params);
}

export async function signInViaOAuth(options: SignInViaOAuthOptions): Promise<string> {
  return await auth.signInViaOAuth(options);
}

export async function signOut(): Promise<void> {
  try {
    await requireAuth();
    const url = await getSignOutUrl();
    if (url) {
      redirect(url);
    }
    else redirect("/auth/sign-in");
  }
  catch(error) {
    console.error("Failed to get sign out url:", error);
    redirect("/auth/sign-in");
  }
  finally {
    await deleteCookie(UNKEY_SESSION_COOKIE);
  }
}

export async function inviteMember(params: OrgInvite): Promise<Invitation> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.inviteMember(params);
}

export async function getOrg(orgId: string): Promise<Organization> {
  const user = await requireAuth();
  await requireOrgAccess(orgId, user.id);
  return await auth.getOrg(orgId);
}

export async function getWorkspace(tenantId: string): Promise<any> {
  if (!tenantId) {
    throw new Error("TenantId/orgId is required to look up workspace");
  }
  const user = await requireAuth();
  // Only allow users to retrieve their workspace
  if (tenantId !== user.orgId) {
    throw new Error('Unauthorized to view other users memberships');
  }
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  return workspace;
}

export async function getOrganizationMemberList(orgId: string): Promise<OrgMembership> {
  if (!orgId) {
    throw new Error("OrgId is required.");
  }
  const user = await requireAuth();
  await requireOrgAdmin(orgId, user.id);
  return await auth.getOrganizationMemberList(orgId);
}

export async function getInvitationList(orgId: string): Promise<OrgInvitation> {
  if (!orgId) {
    throw new Error("OrgId is required.");
  }
  const user = await requireAuth();
  await requireOrgAdmin(orgId, user.id);
  return await auth.getInvitationList(orgId);
}

export async function removeMembership(params: {membershipId: string, orgId: string}): Promise<Invitation> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.removeMembership(params.membershipId);
}

export async function updateMembership(params: {membershipId: string, orgId: string, role: string}): Promise<Invitation> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.updateMembership({membershipId: params.membershipId, role: params.role});
}

export async function revokeOrgInvitation(params: {invitationId: string, orgId: string}): Promise<Invitation> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.revokeOrgInvitation(params.invitationId);
}