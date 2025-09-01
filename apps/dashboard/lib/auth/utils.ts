"use server";

import { redirect } from "next/navigation";
import { getAuthOrRedirect } from "../auth";
import { deleteCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_SESSION_COOKIE } from "./types";

// Helper function for ensuring a signed-in user
export async function requireAuth(): Promise<{
  userId: string | null;
  orgId: string | null;
}> {
  const authResult = await getAuthOrRedirect();
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
  await requireAuth();
  //const signOutUrl = await auth.getSignOutUrl();
  await deleteCookie(UNKEY_SESSION_COOKIE);
  redirect("/auth/sign-in");
}
