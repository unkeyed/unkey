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
  OrgInvitation,
  AuthErrorCode
} from './types';
import { deleteCookie, setCookies } from './cookies';
import { redirect } from 'next/navigation';
import { db } from '../db';
import { SIGN_IN_URL } from '@clerk/nextjs/server';

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


/*
  * Sign out the current user and redirect to the sign in page.
  * This function will delete the session cookie before redirecting to the sign in page
  * @returns {Promise<void>}
*/
export async function signOut(): Promise<void> {
  let redirectPath: string | null = null
    await requireAuth();
    redirectPath = await getSignOutUrl();
    
    if (!redirectPath) {
      redirectPath = "/auth/sign-in";
    }

  // Always delete the session cookie before redirecting to sign out
    await deleteCookie(UNKEY_SESSION_COOKIE);
    redirect(redirectPath || "/auth/sign-in");
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

export async function signUpViaEmail(params: {firstName: string, lastName: string, email: string}): Promise<any> {
  // public
  return await auth.signUpViaEmail(params);
}

export async function signInViaEmail(email:string) {
  return await auth.signInViaEmail(email);
}

export async function verifyAuthCode(params: {email: string, code: string}) {
  try {
    const result = await auth.verifyAuthCode(params);
    
    if (result.success) {
      if (result.cookies?.length) {
        await setCookies(result.cookies);
      }
      
      // Redirect on success
      redirect(result.redirectTo);
    }
    
    // Return the error result if verification failed
    return result;
    
  } catch (error) {
    console.error('Auth code verification failed:', error);
    return {
      success: false,
      redirectTo: SIGN_IN_URL,
      cookies: [],
      error: error instanceof Error ? error : new Error(AuthErrorCode.UNKNOWN_ERROR)
    };
  }
}

export async function resendAuthCode(email: string) {
  if (!email) {
    throw new Error("Email address is required.");
  }
  // TODO
  return;
}