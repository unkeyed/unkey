import { setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies";
import { getAuth as getBaseAuth } from "@/lib/auth/get-auth";
import { localAuth } from "@/lib/auth/local";
import { auth } from "@/lib/auth/server";
import type { AuthenticatedUser } from "@/lib/auth/types";
import { env, workosAuthEnv } from "@/lib/env";
import { routes } from "@/lib/navigation/routes";
import type { Route } from "next";
import { headers } from "next/headers";
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
 * Reads the AuthKit session in WorkOS mode or the deterministic session in
 * local mode, then enforces the dashboard's user and organization requirements.
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
 * Retrieves the provider-backed user and combines it with session organization
 * and impersonation details.
 */
export async function getCurrentUser(): Promise<AuthenticatedUser> {
  const { userId, orgId, impersonator, role } = await getAuth();

  const user = await auth.getUser(userId); // getAuth will redirect if there's no userId
  if (!user) {
    redirect("/auth/sign-in" as Route);
  }
  return { ...user, orgId, role, impersonator };
}

/**
 * Switches the user's active organization and persists the refreshed session.
 *
 * The document-navigation route calls this helper so AuthKit can propagate
 * required SSO or MFA redirects directly to the browser.
 *
 * @param organizationId - The organization ID to switch to
 * @throws Error if the auth provider cannot issue a session for the org
 */
export async function switchToOrg(organizationId: string): Promise<void> {
  if (env().AUTH_PROVIDER === "workos") {
    workosAuthEnv();
    const { switchToOrganization } = await import("@workos-inc/authkit-nextjs");
    await switchToOrganization(organizationId, { revalidationStrategy: "none" });
  } else {
    const { newToken, expiresAt } = await localAuth.switchOrg(organizationId);
    if (!newToken || !expiresAt) {
      throw new Error("Invalid session data returned from auth provider");
    }
    await setSessionCookie({ token: newToken, expiresAt });
  }

  try {
    await setLastUsedOrgCookie({ orgId: organizationId });
  } catch (_error) {
    // The switch itself succeeded. This cookie only preselects the org on the
    // next sign-in, so losing it must not fail the switch.
  }
}
