"use server";

import type { Route } from "next";
import { redirect } from "next/navigation";
import { deleteCookie, getCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_SESSION_COOKIE } from "./types";

// Sign Out
export async function signOut(): Promise<void> {
  const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
  if (sessionToken) {
    await auth.revokeSession(sessionToken);
  }
  await deleteCookie(UNKEY_SESSION_COOKIE);
  redirect("/auth/sign-in" as Route);
}
