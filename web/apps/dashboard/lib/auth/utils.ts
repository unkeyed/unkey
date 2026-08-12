"use server";

import type { Route } from "next";
import { redirect } from "next/navigation";
import { env, workosAuthEnv } from "../env";
import { deleteCookie } from "./cookies";
import { UNKEY_SESSION_COOKIE } from "./types";

// Sign Out
export async function signOut(): Promise<void> {
  if (env().AUTH_PROVIDER === "workos") {
    workosAuthEnv();
    const { signOut: signOutWithAuthKit } = await import("@workos-inc/authkit-nextjs");
    await signOutWithAuthKit({ returnTo: "/auth/sign-in" });
    return;
  }

  await deleteCookie(UNKEY_SESSION_COOKIE);
  redirect("/auth/sign-in" as Route);
}
