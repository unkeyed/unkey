"use server";

import { env } from "@/lib/env";
import { redirect } from "next/navigation";
import { getAuth } from "../auth";
import { deleteCookie, getCookie } from "./cookies";
import { auth } from "./server";
import { BETTER_AUTH_SESSION_COOKIE, UNKEY_SESSION_COOKIE } from "./types";

// Helper function for ensuring a signed-in user
export async function requireAuth(): Promise<{
  userId: string | null;
  orgId: string | null;
}> {
  const authResult = await getAuth();
  if (!authResult.userId) {
    redirect("/auth/sign-in");
  }
  return authResult;
}

// Helper to check invite email matches
export async function requireEmailMatch(params: {
  email: string;
  invitationToken: string;
}): Promise<void> {
  const { email, invitationToken } = params;
  try {
    const invitation = await auth.getInvitation(invitationToken);
    if (invitation?.email !== email) {
      throw new Error("Email address does not match the invitation email.");
    }
  } catch (_error) {
    throw new Error("Invalid invitation");
  }
}

// Sign Out
export async function signOut(): Promise<void> {
  const config = env();

  if (config.AUTH_PROVIDER === "better-auth") {
    // Get the session token to revoke it server-side
    const sessionToken = await getCookie(BETTER_AUTH_SESSION_COOKIE);

    if (sessionToken) {
      // Revoke the session server-side using Better Auth API
      const { getBetterAuthInstance } = await import("./better-auth-server");
      const betterAuth = getBetterAuthInstance();

      try {
        await betterAuth.api.revokeSession({
          body: { token: sessionToken },
          headers: { cookie: `better-auth.session_token=${sessionToken}` },
        });
      } catch (_error) {
        // Continue with cookie deletion even if revocation fails
      }
    }

    // Delete the Better Auth session cookie
    await deleteCookie(BETTER_AUTH_SESSION_COOKIE);
  }

  // Always delete the Unkey session cookie (used by WorkOS and local providers)
  await deleteCookie(UNKEY_SESSION_COOKIE);

  redirect("/auth/sign-in");
}
