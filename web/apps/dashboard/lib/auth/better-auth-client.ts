"use client";

import { createAuthClient } from "better-auth/react";

/**
 * Better Auth client for frontend usage.
 *
 * This client is used for client-side authentication operations like
 * OAuth sign-in with social providers. It communicates with the
 * Better Auth API routes at /api/auth/*.
 */
export const betterAuthClient = createAuthClient({
  baseURL: typeof window !== "undefined" ? window.location.origin : "",
});

/**
 * Initiates OAuth sign-in with a social provider using Better Auth client.
 * This will redirect the user to the OAuth provider automatically.
 *
 * @param provider - The OAuth provider (github or google)
 * @param callbackURL - URL to redirect after successful authentication
 */
export async function signInWithSocial(
  provider: "github" | "google",
  callbackURL: string,
): Promise<void> {
  await betterAuthClient.signIn.social({
    provider,
    callbackURL,
  });
}
