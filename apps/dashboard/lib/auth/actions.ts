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
  Organization
} from './types';
import { deleteCookie } from './cookies';
import { redirect } from 'next/navigation';

export async function getCurrentUser(): Promise<User | null> {
  return await auth.getCurrentUser();
}

export async function listMemberships(userId?: string): Promise<OrgMembership> {
  return await auth.listMemberships(userId);
}

export async function refreshSession(orgId: string): Promise<void> {
  return await auth.refreshSession(orgId);
}

export async function getSignOutUrl(): Promise<string | null> {
  const url = await auth.getSignOutUrl();
  return url;
}

export async function createTenant(params: { name: string, userId: string }): Promise<string> {
  return await auth.createTenant(params);
}

export async function signInViaOAuth(options: SignInViaOAuthOptions): Promise<string> {
  return await auth.signInViaOAuth(options);
}

export async function signOut(): Promise<void> {
  try {
    const url = await getSignOutUrl();
    if (url) {
      redirect(url)
    }
    else redirect("/auth/sign-in")
  }

  catch(error) {
    console.error("Failed to get sign out url:", error);
    redirect("/auth/sign-in")
  }
  finally {
    await deleteCookie(UNKEY_SESSION_COOKIE);
  }
}

export async function inviteMember(params: OrgInvite): Promise<Invitation> {
  return await auth.inviteMember(params);
}

export async function getOrg(orgId: string): Promise<Organization> {
  return await auth.getOrg(orgId);
}