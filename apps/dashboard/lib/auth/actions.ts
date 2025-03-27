"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { db, Quotas, Workspace } from "../db";
import { deleteCookie, getCookie, setCookie, setCookies } from "./cookies";
import { auth } from "./server";
import {
  AuthErrorCode,
  type AuthErrorResponse,
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  type Membership,
  type MembershipListResponse,
  type NavigationResponse,
  type OAuthResult,
  type OrgInviteParams,
  type Organization,
  PENDING_SESSION_COOKIE,
  type SignInViaOAuthOptions,
  UNKEY_SESSION_COOKIE,
  type User,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "./types";

// Helper function to check authentication
async function requireAuth(): Promise<User> {
  const user = await auth.getCurrentUser();
  if (!user) {
    redirect("/auth/sign-in");
  }
  return user;
}

// Helper function to check organization access
async function requireOrgAccess(orgId: string, userId: string): Promise<void> {
  const memberships = await auth.listMemberships(userId);
  const hasAccess = memberships.data.some((m) => m.organization.id === orgId);
  if (!hasAccess) {
    throw new Error("You do not have access to this organization");
  }
}

// Helper to check admin status
async function requireOrgAdmin(orgId: string, userId: string): Promise<void> {
  const memberships = await auth.listMemberships(userId);
  const isAdmin = memberships.data.some((m) => m.organization.id === orgId && m.role === "admin");
  if (!isAdmin) {
    throw new Error("This action requires admin privileges");
  }
}

// Helper to check invite email matches
async function requireEmailMatch(params: {
  email: string;
  invitationToken: string;
}): Promise<void> {
  const { email, invitationToken } = params;
  try {
    const invitation = await auth.getInvitation(invitationToken);
    if (invitation?.email !== email) {
      throw new Error("Email address does not match the invitation email.");
    }
  } catch (error) {
    throw new Error("Invalid invitation");
  }
}

// Authentication Actions
export async function signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
  return await auth.signUpViaEmail(params);
}

export async function signInViaEmail(email: string): Promise<EmailAuthResult> {
  return await auth.signInViaEmail(email);
}

export async function verifyAuthCode(params: {
  email: string;
  code: string;
  invitationToken?: string;
}): Promise<VerificationResult> {
  const { email, code, invitationToken } = params;
  try {
    if (invitationToken) {
      await requireEmailMatch({ email, invitationToken });
    }

    const result = await auth.verifyAuthCode({ email, code, invitationToken });

    if (result.cookies) {
      await setCookies(result.cookies);
    }

    return result;
  } catch (error) {
    console.error(error);
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }
}

export async function verifyEmail(code: string): Promise<VerificationResult> {
  try {
    // get the pending auth token
    // it's only good for 10 minutes
    const token = await getCookie(PENDING_SESSION_COOKIE);

    if (!token) {
      console.error("Pending auth token missing or expired");
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
      };
    }

    const result = await auth.verifyEmail({ code, token });

    if (result.cookies) {
      await setCookies(result.cookies);
    }

    return result;
  } catch (error) {
    console.error(error);
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }
}

export async function resendAuthCode(email: string): Promise<EmailAuthResult> {
  if (!email.trim()) {
    return {
      success: false,
      code: AuthErrorCode.INVALID_EMAIL,
      message: "Email address is required.",
    };
  }
  return await auth.resendAuthCode(email);
}

export async function signIntoWorkspace(orgId: string): Promise<VerificationResult> {
  const pendingToken = cookies().get("sess-temp")?.value;

  if (!pendingToken) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: "No pending authentication found",
    };
  }

  try {
    const result = await auth.completeOrgSelection({
      orgId,
      pendingAuthToken: pendingToken,
    });

    if (result.success) {
      await setCookies(result.cookies);
      await deleteCookie("sess-temp");
      redirect(result.redirectTo);
    }

    return result;
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: error instanceof Error ? error.message : "Unknown error occurred",
    };
  }
}

// User & Session Management
export async function getCurrentUser(): Promise<User | null> {
  return await auth.getCurrentUser();
}

export async function listMemberships(userId: string): Promise<MembershipListResponse> {
  await requireAuth();
  return await auth.listMemberships(userId);
}

export async function switchOrg(orgId: string): Promise<{ success: boolean; error?: string }> {
  const user = await requireAuth();
  await requireOrgAccess(orgId, user.id);
  if (!orgId) {
    return { success: false, error: "Missing organization ID" };
  }

  try {
    const { newToken, expiresAt } = await auth.switchOrg(orgId);

    // Set the new cookie
    await setCookie({
      name: UNKEY_SESSION_COOKIE,
      value: newToken,
      options: {
        httpOnly: true,
        secure: true,
        sameSite: "lax",
        path: "/",
        maxAge: Math.floor((expiresAt.getTime() - Date.now()) / 1000),
      },
    });

    return { success: true };
  } catch (error) {
    console.error("Organization switch failed:", error);
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to switch organization",
    };
  }
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
      message: error instanceof Error ? error.message : "Unknown error occurred",
    };
  }
}

// Organization Selection
export async function completeOrgSelection(
  orgId: string,
): Promise<NavigationResponse | AuthErrorResponse> {
  const tempSession = cookies().get(PENDING_SESSION_COOKIE);
  if (!tempSession) {
    throw new Error("No pending session");
  }

  // Call auth provider with token and orgId
  const result = await auth.completeOrgSelection({ pendingAuthToken: tempSession.value, orgId });

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
  redirect(signOutUrl || "/auth/sign-in");
}

// Organization Management
export async function createTenant(params: { name: string; userId: string }): Promise<string> {
  const user = await requireAuth();
  if (params.userId !== user.id) {
    throw new Error("Unauthorized to create tenant for another user");
  }
  return await auth.createTenant(params);
}

export async function getWorkspace(orgId: string): Promise<Workspace & { quota: Quotas }> {
  if (!orgId) {
    throw new Error("TenantId/orgId is required to look up workspace");
  }
  const user = await requireAuth();
  if (orgId !== user.orgId) {
    throw new Error("Unauthorized to view other users memberships");
  }
  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: {
      quota: true,
    },
  });
  // Quotas are non-optional, we just can't tell drizzle that they are
  return ws as Workspace & { quota: Quotas };
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
  console.log("here");
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

export async function removeMembership(params: {
  membershipId: string;
  orgId: string;
}): Promise<void> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.removeMembership(params.membershipId);
}

export async function updateMembership(params: {
  membershipId: string;
  orgId: string;
  role: string;
}): Promise<Membership> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.updateMembership({ membershipId: params.membershipId, role: params.role });
}

export async function revokeOrgInvitation(params: {
  invitationId: string;
  orgId: string;
}): Promise<void> {
  const user = await requireAuth();
  await requireOrgAdmin(params.orgId, user.id);
  return await auth.revokeOrgInvitation(params.invitationId);
}
