'use server';

import { auth } from './index';
import { OrgMembership, UNKEY_SESSION_COOKIE, OAuthStrategy } from './interface';
import { deleteCookie } from './cookies';

export async function listMembershipsAction(userId?: string): Promise<OrgMembership> {
  return await auth.listMemberships(userId);
}

export async function refreshSessionAction(orgId: string): Promise<void> {
  await auth.refreshSession(orgId);
}

export async function getCurrentUserAction() {
  return await auth.getCurrentUser();
}

export async function getSignOutUrlAction() {
  const url = await auth.getSignOutUrl();
  await deleteCookie(UNKEY_SESSION_COOKIE);
  return url;
}

export async function initiateOAuthSignIn({
  provider, 
  redirectUrlComplete
}: {
  provider: OAuthStrategy;
  redirectUrlComplete: string;
}): Promise<{ url: string | null; error?: string }> {
  try {
    const url = auth.signInViaOAuth({ 
      provider,
      redirectUrlComplete
    });
    return { url };
  } catch (error) {
    console.error('OAuth initialization error:', error);
    return { 
      url: null, 
      error: error instanceof Error ? error.message : 'Authentication failed'
    };
  }
}