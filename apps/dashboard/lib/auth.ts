import { getAuth as getBaseAuth } from "@/lib/auth/get-auth";
import { auth } from "@/lib/auth/server";
import type { AuthenticatedUser } from "@/lib/auth/types";
import { redirect } from "next/navigation";
import type { NextRequest } from "next/server";

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
    redirect("/auth/sign-in");
  }

  if (!authResult.orgId && !authResult.role) {
    redirect("/new");
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
    redirect("/auth/sign-in");
  }
  return { ...user, orgId, role, impersonator };
}

/**
 * Handles invitation acceptance and organization switching for authenticated users.
 *
 * This function accepts an invitation and switches the user's active organization
 * to the invited organization. It should be used when a user is already authenticated
 * and clicking on an invitation link.
 *
 * @param invitationId - The ID of the invitation to accept
 * @param organizationId - The organization ID to switch to
 * @returns Promise that resolves when the invitation is accepted and org is switched
 * @throws Error if invitation acceptance or org switching fails
 */
export async function acceptInvitationAndSwitchOrg(
  invitationId: string,
  organizationId: string,
): Promise<void> {
  try {
    // Verify we have a valid session before proceeding
    const currentUser = await getCurrentUser();
    if (!currentUser) {
      throw new Error("User not authenticated - cannot accept invitation");
    }

    // Accept the invitation first
    await auth.acceptInvitation(invitationId);

    // Then switch to the organization
    const { switchOrg } = await import("@/app/auth/actions");
    const result = await switchOrg(organizationId);

    if (!result.success) {
      throw new Error(result.error || "Failed to switch organization");
    }
  } catch (error) {
    console.error("Failed to accept invitation and switch org:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    throw error;
  }
}

/**
 * Handles invitation processing after successful authentication.
 *
 * This function should be called after a user successfully authenticates
 * when they came from an invitation link. It will accept the invitation
 * and switch their active organization if needed.
 *
 * @param invitationToken - The invitation token from the URL
 * @param userId - The authenticated user's ID
 * @returns Promise that resolves when invitation is processed
 */
export async function processPostAuthInvitation(
  invitationToken: string,
  userId: string,
): Promise<{ success: boolean; organizationId?: string; error?: string }> {
  try {
    // Get the invitation details
    const invitation = await auth.getInvitation(invitationToken);

    if (!invitation) {
      return { success: false, error: "Invitation not found" };
    }

    const { email: invitationEmail, state, organizationId, id: invitationId } = invitation;

    if (state !== "pending") {
      return { success: false, error: `Invitation is ${state}` };
    }

    if (!organizationId) {
      return { success: false, error: "No organization ID in invitation" };
    }

    // Get the user to verify email matches
    const user = await auth.getUser(userId);
    if (!user) {
      return { success: false, error: "User not found" };
    }

    if (user.email !== invitationEmail) {
      return { success: false, error: "Email mismatch" };
    }

    // Accept the invitation and switch organization
    await acceptInvitationAndSwitchOrg(invitationId, organizationId);

    return { success: true, organizationId };
  } catch (error) {
    console.error("Failed to process post-auth invitation:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return {
      success: false,
      error: error instanceof Error ? error.message : "Unknown error",
    };
  }
}
