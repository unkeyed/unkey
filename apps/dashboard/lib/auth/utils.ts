"use server";

import { redirect } from "next/navigation";
import { deleteCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_SESSION_COOKIE, type User } from "./types";

// Helper function for ensuring a signed-in user
export async function requireAuth(): Promise<User> {
  const user = await auth.getCurrentUser();
  if (!user) {
    redirect("/auth/sign-in");
  }
  return user;
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
