"use server";

import { redirect } from "next/navigation";
import { deleteCookie, getCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_SESSION_COOKIE } from "./types";

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
  const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
  if (sessionToken) {
    await auth.revokeSession(sessionToken);
  }
  await deleteCookie(UNKEY_SESSION_COOKIE);
  redirect("/auth/sign-in");
}
