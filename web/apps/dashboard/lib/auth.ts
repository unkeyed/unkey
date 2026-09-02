import type { Route } from "next";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import type { NextRequest } from "next/server";
import { setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies";
import { getAuth as getBaseAuth } from "@/lib/auth/get-auth";
import { auth } from "@/lib/auth/server";
import type { AuthenticatedUser } from "@/lib/auth/types";
import { routes } from "@/lib/navigation/routes";

type GetAuthResult = {
  userId: string;
  orgId: string;
  role: string;
  impersonator?: {
    email: string;
    reason?: string | null;
  };
};

/**
 * Validates the current user session and performs token refresh if needed.
 *
 * This function checks for a valid authentication cookie, validates the session,
 * and handles token refreshing if the current token is expired but refreshable.
 * Results are cached for the duration of the server request to prevent
 * multiple validation calls.
 *
 * @param _req - Optional request object (not used but maintained for compatibility)
 * @returns Authentication result containing userId and orgId if authenticated, null values otherwise
 * @throws Redirects to sign-in or organization/workspace creation pages if requirements aren't met
 */
export async function getAuth(req?: NextRequest): Promise<GetAuthResult> {
  const authResult = await getBaseAuth(req);
  if (!authResult.userId) {
    // Read the current path from the custom header set by the middleware (proxy.ts)
    let signInUrl = "/auth/sign-in";
    try {
      const headersList = await headers();
      const currentPath = headersList.get("x-current-path");
      if (currentPath && currentPath !== "/") {
        signInUrl = `/auth/sign-in?redirect=${encodeURIComponent(currentPath)}`;
      }
    } catch {
      // Ignore header read errors
    }
    redirect(signInUrl as Route);
  }

  if (!authResult.orgId && !authResult.role) {
    redirect(routes.workspaces.create());
  }

  return authResult as GetAuthResult;
}

/**
 * Retrieves the complete current user object with organization information.
 *
 * This function fetches the authenticated user from the database along with
 * their organization ID. It will redirect to the sign-in page if the user
 * is not authenticated or cannot be found in the database.
 * Results are cached for the duration of the server request.
 *
 * @returns Full user object with organization ID
 * @throws Redirects to sign-in page if user is not authenticated or not found
 */
export async function getCurrentUser(): Promise<AuthenticatedUser> {
  const { userId, orgId, impersonator, role } = await getAuth();

  const user = await auth.getUser(userId); // getAuth will redirect if there's no userId
  if (!user) {
    redirect("/auth/sign-in" as Route);
  }
  return { ...user, orgId, role, impersonator };
}

// The only two failure messages the invitation flow may show a user. They are
// fixed literals because ENG-3014 was a raw provider error reaching the
// client, and every failure below funnels into one of these instead. They are
// also deliberately coarse: distinguishing "no such token" from "revoked"
// would let any signed-in user probe arbitrary tokens for validity.
const INVITATION_UNUSABLE = "This invitation is no longer valid. Ask for a new one.";
const EMAIL_MISMATCH = "This invitation was sent to a different email address.";

/**
 * Invitation emails and stored account emails can differ in casing or
 * whitespace (e.g. invited as User@Example.com, account stored lowercased),
 * so every invitation email check must compare the normalized form.
 */
export function normalizeEmail(email: string): string {
  return email.trim().toLowerCase();
}

/**
 * Switches the user's active organization and persists the refreshed session.
 *
 * Server-side counterpart to the user.switchOrg tRPC mutation the org picker
 * uses; lives in lib/ (not app/auth/actions.ts) so lib/ never statically
 * imports a "use server" module from app/ — a module cycle through a
 * server-action boundary surfaces as undefined exports at runtime rather
 * than as a build error.
 *
 * @param organizationId - The organization ID to switch to
 * @throws Error if the auth provider cannot issue a session for the org
 */
async function switchToOrg(organizationId: string): Promise<void> {
  const { newToken, expiresAt } = await auth.switchOrg(organizationId);
  if (!newToken || !expiresAt) {
    throw new Error("Invalid session data returned from auth provider");
  }
  await setSessionCookie({ token: newToken, expiresAt });
  try {
    await setLastUsedOrgCookie({ orgId: organizationId });
  } catch (_error) {
    // The switch itself succeeded. This cookie only preselects the org on the
    // next sign-in, so losing it must not fail the switch.
  }
}

/**
 * Outcome of an invitation acceptance attempt.
 *
 * `switched: false` means the invitation was accepted — the user IS a member
 * of the org — but the active-org switch failed. Callers must surface that
 * distinction: reporting it as a plain success drops the user back into their
 * previous org (or, for a first-org signup, into workspace creation) with no
 * explanation and no invitation left to retry.
 *
 * Every `error` is a fixed, user-safe literal (never a raw provider message),
 * so callers may surface it directly.
 */
export type InvitationResult =
  | { success: true; organizationId: string; switched: boolean }
  | { success: false; error: string };

/**
 * Accepts an invitation for an authenticated user and switches their active
 * organization to it.
 *
 * The caller must have validated (and, if needed, refreshed) the session via
 * a request-less getAuth() first, so the cookie store holds a session the org
 * switch can consume: acceptInvitation is a session-independent API-key call
 * that irreversibly consumes the invitation, and it must not run against a
 * session that cannot complete the switch. A request-bound updateSession does
 * NOT satisfy this — it writes the refreshed cookie to a Headers object the
 * caller usually discards, leaving auth.switchOrg to read a stale session
 * whose refresh token was already spent.
 *
 * Only "pending" invitations are processed. An already-accepted invitation is
 * inert on purpose: a stale link must never silently move an active user out
 * of the org they are working in.
 *
 * @param invitationToken - The invitation token from the URL
 * @param userId - The authenticated user's ID, already validated by the caller
 */
export async function processPostAuthInvitation(
  invitationToken: string,
  userId: string,
): Promise<InvitationResult> {
  try {
    // Get the invitation details
    const invitation = await auth.getInvitation(invitationToken);

    if (!invitation) {
      return { success: false, error: INVITATION_UNUSABLE };
    }

    const { email: invitationEmail, state, organizationId, id: invitationId } = invitation;

    if (!organizationId) {
      return { success: false, error: INVITATION_UNUSABLE };
    }

    if (state !== "pending") {
      // Deliberately the same message as a missing or unknown token, and
      // checked before the email comparison so a non-pending token never leaks
      // an EMAIL_MISMATCH signal. The state comes from the provider and must
      // not be echoed back, and telling a caller which arbitrary tokens are
      // real would make this an oracle.
      return { success: false, error: INVITATION_UNUSABLE };
    }

    // Get the user to verify email matches
    const user = await auth.getUser(userId);
    if (!user) {
      return { success: false, error: INVITATION_UNUSABLE };
    }

    if (normalizeEmail(user.email) !== normalizeEmail(invitationEmail)) {
      return { success: false, error: EMAIL_MISMATCH };
    }

    // Membership exists from here on, even if the switch below fails.
    await auth.acceptInvitation(invitationId);

    try {
      await switchToOrg(organizationId);
    } catch (error) {
      // The user has joined the org and only the active-org switch failed,
      // either transiently or because the org enforces an MFA policy this
      // flow cannot satisfy. The invitation is already spent, so failing here
      // would be a dead end. Report the partial outcome instead and let the
      // caller tell the user to switch workspaces manually.
      console.error("Invitation accepted but org switch failed:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return { success: true, organizationId, switched: false };
    }

    return { success: true, organizationId, switched: true };
  } catch (error) {
    console.error("Failed to process post-auth invitation:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return { success: false, error: INVITATION_UNUSABLE };
  }
}
