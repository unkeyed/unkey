import { getAuth as getBaseAuth } from "@/lib/auth/get-auth";
import { auth } from "@/lib/auth/server";
import type { AuthenticatedUser } from "@/lib/auth/types";
import { redirect } from "next/navigation";
import type { NextRequest } from "next/server";

export type GetAuthResult = {
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
export async function getAuthOrRedirect(req?: NextRequest): Promise<GetAuthResult> {
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
  const { userId, orgId, impersonator, role } = await getAuthOrRedirect();

  const user = await auth.getUser(userId); // getAuth will redirect if there's no userId
  if (!user) {
    redirect("/auth/sign-in");
  }
  return { ...user, orgId, role, impersonator };
}
