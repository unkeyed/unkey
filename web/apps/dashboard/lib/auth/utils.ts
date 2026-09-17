"use server";

import type { Route } from "next";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import { env, workosAuthEnv } from "../env";
import { getBaseUrl } from "../utils";
import { deleteCookie } from "./cookies";
import { UNKEY_SESSION_COOKIE } from "./types";

async function getSignOutReturnTo(): Promise<string> {
  const requestUrl = (await headers()).get("x-url");
  const baseUrl = requestUrl ?? getBaseUrl();
  return new URL("/auth/sign-in", baseUrl).toString();
}

// Sign Out
export async function signOut(): Promise<void> {
  if (env().AUTH_PROVIDER === "workos") {
    workosAuthEnv();
    const { signOut: signOutWithAuthKit } = await import("@workos-inc/authkit-nextjs");
    await signOutWithAuthKit({ returnTo: await getSignOutReturnTo() });
    return;
  }

  await deleteCookie(UNKEY_SESSION_COOKIE);
  redirect("/auth/sign-in" as Route);
}
