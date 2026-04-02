"use server";

import { env } from "@/lib/env";
import { getCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_SESSION_COOKIE } from "./types";

/**
 * Server action that retrieves the JWT access token from the current session.
 * This token can be sent as a Bearer token to the Go API for session-based authentication.
 *
 * In local mode, returns a static token that the local session auth provider accepts.
 */
export async function getAccessToken(): Promise<string | null> {
  const environment = env();

  if (environment.AUTH_PROVIDER === "local") {
    // In local mode, return a dummy token. The Go API's local session auth
    // provider accepts any token.
    return "local_access_token";
  }

  const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
  if (!sessionToken) {
    return null;
  }

  const result = await auth.validateSession(sessionToken);
  if (!result.isValid || !result.accessToken) {
    return null;
  }

  return result.accessToken;
}
