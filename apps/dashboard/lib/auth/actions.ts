'use server';

import { auth } from './server';
import { 
  type User, 
  type MembershipListResponse,
  type EmailAuthResult,
  type VerificationResult,
  type OAuthResult,
  UserData,
  UNKEY_SESSION_COOKIE,
  AuthErrorCode,
  SignInViaOAuthOptions,
  OrgInviteParams,
  Invitation,
  Organization,
  InvitationListResponse,
  UpdateMembershipParams,
  UpdateOrgParams,
  Membership,
  SessionData,
  NavigationResponse,
  PENDING_SESSION_COOKIE,
  AuthErrorResponse,
  errorMessages,
} from './types';
import { cookies } from 'next/headers';
import { Cookie, deleteCookie, setCookie, setCookies } from './cookies';
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
  const memberships = await auth.listMemberships();
  const hasAccess = memberships.data.some(m => m.organization.id === orgId);
  if (!hasAccess) {
    throw new Error('You do not have access to this organization');
  }
}

// Helper to check admin status
async function requireOrgAdmin(orgId: string, userId: string): Promise<void> {
  const memberships = await auth.listMemberships();
  const isAdmin = memberships.data.some(
    m => m.organization.id === orgId && m.role === 'admin'
  );
  if (!isAdmin) {
    throw new Error('This action requires admin privileges');
  }
}

// Authentication Actions
export async function signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
  return await auth.signUpViaEmail(params);
}

export async function signInViaEmail(email: string): Promise<EmailAuthResult> {
  return await auth.signInViaEmail(email);
}

export async function verifyAuthCode(params: { email: string; code: string }): Promise<VerificationResult> {
  const {email, code} = params;
  try {
    const result = await auth.verifyAuthCode({email, code});

    if (result.cookies) {
      await setCookies(result.cookies);
    }

    return result;
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR]
    };
  }
}

export async function resendAuthCode(email: string): Promise<EmailAuthResult> {
  if (!email.trim()) {
    return {
      success: false,
      code: AuthErrorCode.INVALID_EMAIL,
      message: "Email address is required."
    };
  }
  return await auth.resendAuthCode(email);
}

export async function signIntoWorkspace(orgId: string): Promise<VerificationResult> {
  const pendingToken = cookies().get('sess-temp')?.value;
  
  if (!pendingToken) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: "No pending authentication found"
    };
  }

  try {
    const result = await auth.completeOrgSelection({ 
      orgId, 
      pendingAuthToken: pendingToken 
    });

    if (result.success) {
      await setCookies(result.cookies);
      await deleteCookie('sess-temp');
      redirect(result.redirectTo);
    }

    return result;
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

// User & Session Management
export async function getCurrentUser(): Promise<User | null> {
  return await auth.getCurrentUser();
}

export async function listMemberships(): Promise<MembershipListResponse> {
  await requireAuth();
  return await auth.listMemberships();
}

export async function refreshSession(orgId: string): Promise<SessionData | null> {
  const user = await requireAuth();
  await requireOrgAccess(orgId, user.id);
  return await auth.refreshSession(orgId);
}

// OAuth
export async function signInViaOAuth(options: SignInViaOAuthOptions): Promise<string> {
  return await auth.signInViaOAuth(options);
}

export async function completeOAuthSignIn(request: Request): Promise<OAuthResult> {
  try {
    const result = await auth.completeOAuthSignIn(request);
    
    if (result.success) {
      await setCookies(result.cookies);
      redirect(result.redirectTo);
    }

    return result;
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

// Organization Selection
export async function checkPendingAuth(): Promise<boolean> {
  const tempSession = cookies().get(PENDING_SESSION_COOKIE);
  return tempSession !== undefined;
}

export async function completeOrgSelection(orgId: string): Promise<NavigationResponse | AuthErrorResponse> {
  const tempSession = cookies().get(PENDING_SESSION_COOKIE);
  if (!tempSession) throw new Error('No pending session');

  // Call auth provider with token and orgId
  const result = await auth.completeOrgSelection({pendingAuthToken: tempSession.value, orgId});
  
  if (result.success) {
    cookies().delete(PENDING_SESSION_COOKIE);
    for (const cookie of result.cookies) {
      cookies().set(cookie.name, cookie.value, cookie.options);
    }
  }
  
  return result;
}

// Sign Out
export async function signOut(): Promise<void> {
  await requireAuth();
  const signOutUrl = await auth.getSignOutUrl();
  await deleteCookie(UNKEY_SESSION_COOKIE);
  redirect(signOutUrl || '/auth/sign-in');
}

// Organization Management
export async function createTenant(params: { name: string; userId: string }): Promise<string> {
  const user = await requireAuth();
  if (params.userId !== user.id) {
    throw new Error('Unauthorized to create tenant for another user');
  }
  return await auth.createTenant(params);
}

export async function getWorkspace(tenantId: string): Promise<any> {
  if (!tenantId) {
    throw new Error("TenantId/orgId is required to look up workspace");
  }
  const user = await requireAuth();
  if (tenantId !== user.orgId) {
    throw new Error('Unauthorized to view other users memberships');
  }
  return await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
}

export async function getOrg(orgId: string): Promise<Organization> {
  const user = await requireAuth();
  await requireOrgAccess(orgId, user.id);
  return await auth.getOrg(orgId);
}

// Membership & Invitation Management
export async function getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
  if (!orgId) {
    throw new Error("OrgId is required.");
  }
  const user = await requireAuth();
  await requireOrgAdmin(orgId, user.id);
  return await auth.getOrganizationMemberList(orgId);
}

export async function inviteMember(params: OrgInviteParams): Promise<Invitation> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.inviteMember(params);
}

export async function getInvitationList(orgId: string): Promise<InvitationListResponse> {
  if (!orgId) {
    throw new Error("OrgId is required.");
  }
  const user = await requireAuth();
  await requireOrgAdmin(orgId, user.id);
  return await auth.getInvitationList(orgId);
}

export async function removeMembership(params: { membershipId: string; orgId: string }): Promise<void> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.removeMembership(params.membershipId);
}

export async function updateMembership(params: { membershipId: string; orgId: string; role: string }): Promise<Membership> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.updateMembership({ membershipId: params.membershipId, role: params.role });
}

export async function revokeOrgInvitation(params: { invitationId: string; orgId: string }): Promise<void> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.revokeOrgInvitation(params.invitationId);
}
