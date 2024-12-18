"use server"

import { cookies } from 'next/headers';
import { auth } from './index';
import { OrgMembership, UNKEY_SESSION_COOKIE } from './interface';

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
  const cookieStore = cookies();
  cookieStore.delete(UNKEY_SESSION_COOKIE);
  return url;
}