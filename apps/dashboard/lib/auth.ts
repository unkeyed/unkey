import { getAuth as noCacheGetAuth } from "@/lib/auth/get-auth";
import { auth } from "@/lib/auth/server";
import type { User } from "@/lib/auth/types";
import { redirect } from "next/navigation";
import { cache } from "react";

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
};

export async function getIsImpersonator(): Promise<boolean> {
  const user = await auth.getCurrentUser();
  if (!user) {
    return false;
  }
  return user.impersonator !== undefined;
}

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
export const getAuth = cache(async (_req?: Request): Promise<GetAuthResult> => {
  const authResult = await noCacheGetAuth();
  if (!authResult.userId) {
    redirect("/auth/sign-in");
  }

  return authResult;
});

/**
 * Retrieves the current organization ID or redirects if unavailable.
 *
 * This function checks authentication status and organization membership.
 * It will redirect to the sign-in page if the user is not authenticated,
 * or to the workspace creation page if the user has no organization.
 * Results are cached for the duration of the server request.
 *
 * @returns The current user's organization ID
 */
export const getOrgId = cache(async (): Promise<string> => {
  const { orgId } = await getAuth();

  if (!orgId) {
    redirect("/new");
  }

  return orgId;
});

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
export const getCurrentUser = cache(async (): Promise<User> => {
  const { userId, orgId } = await getAuth();

  const user = await auth.getUser(userId!); // getAuth will redirect if there's no userId
  if (!user) {
    redirect("/auth/sign-in");
  }
  return { ...user, orgId }
});

