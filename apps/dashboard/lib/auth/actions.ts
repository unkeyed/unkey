'use server';

import { auth } from './index';
import { OrgMembership } from './interface';

export async function listMembershipsAction(userId?: string): Promise<OrgMembership> {
  return auth.listMemberships(userId);
}

export async function refreshSessionAction(orgId: string): Promise<void> {
  return auth.refreshSession(orgId);
}

export async function getCurrentUserAction() {
  return auth.getCurrentUser();
}