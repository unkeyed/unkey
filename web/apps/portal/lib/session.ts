"use server";

import { env } from "@/lib/env";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const SESSION_COOKIE_NAME = "portal_session";
const SESSION_COOKIE_MAX_AGE_SECONDS = 24 * 60 * 60; // 24 hours

type ExchangeResult =
  | { success: true }
  | { success: false; error: string };

/**
 * Exchange a short-lived session ID for a long-lived browser session token.
 * Sets an httpOnly cookie and redirects to /keys on success.
 */
export async function exchangeSession(sessionId: string): Promise<ExchangeResult> {
  const apiUrl = env().UNKEY_API_URL;

  const response = await fetch(`${apiUrl}/v2/portal.exchangeSession`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sessionId }),
  });

  if (!response.ok) {
    return {
      success: false,
      error: "Session expired or invalid. Please request a new session from your application.",
    };
  }

  const data: { token: string; expiresAt: number } = await response.json();

  const cookieStore = await cookies();
  cookieStore.set(SESSION_COOKIE_NAME, data.token, {
    httpOnly: true,
    secure: true,
    sameSite: "strict",
    path: "/",
    maxAge: SESSION_COOKIE_MAX_AGE_SECONDS,
  });

  redirect("/keys");
}

/**
 * Read the portal session token from the cookie.
 * Returns null if no valid session cookie exists.
 */
export async function getSessionToken(): Promise<string | null> {
  const cookieStore = await cookies();
  return cookieStore.get(SESSION_COOKIE_NAME)?.value ?? null;
}

/**
 * Clear the portal session cookie.
 */
export async function clearSession(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(SESSION_COOKIE_NAME);
}
