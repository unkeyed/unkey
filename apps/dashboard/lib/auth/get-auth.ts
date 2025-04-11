import { cache } from "react";
import { env } from "../env";
import { getCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_REFRESH_TOKEN, UNKEY_SESSION_COOKIE } from "./types";

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
  role: string | null;
};

// Cache session validation based on session token
export const validateSessionCached = cache(
  async (
    sessionToken: string | null,
  ): Promise<{
    isValid: boolean;
    shouldRefresh: boolean;
    userId?: string;
    orgId?: string | null;
    role?: string | null;
  }> => {
    if (!sessionToken) {
      return { isValid: false, shouldRefresh: false };
    }

    try {
      return await auth.validateSession(sessionToken);
    } catch (error) {
      console.error("Session validation error:", error);
      return { isValid: false, shouldRefresh: false };
    }
  },
);

// Global mutex objects to prevent concurrent refresh operations
let refreshInProgress = false;
let refreshPromise: Promise<GetAuthResult> | null = null;

// refresh token with mutex protection
async function refreshTokenWithMutex(
  refreshToken: string,
  baseUrl: string,
): Promise<GetAuthResult> {
  // If refresh is already in progress, wait for it
  if (refreshInProgress && refreshPromise) {
    try {
      return await refreshPromise;
    } catch (error) {
      console.error("Error while waiting for refresh:", error);
      return { userId: null, orgId: null, role: null };
    }
  }

  // Set mutex to prevent concurrent refreshes
  refreshInProgress = true;

  // Create the refresh promise
  refreshPromise = (async (): Promise<GetAuthResult> => {
    try {
      const refreshResult = await fetch(`${baseUrl}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: {
          "x-refresh-token": refreshToken,
        },
      });

      if (!refreshResult.ok) {
        throw new Error("Refresh failed");
      }

      const refreshedData = await refreshResult.json();

      return {
        userId: refreshedData.session.userId,
        orgId: refreshedData.session.orgId,
        role: refreshedData.session.role,
      };
    } catch (error) {
      console.error("Refresh error:", error);
      return { userId: null, orgId: null, role: null };
    } finally {
      // Always clear the mutex when done
      refreshInProgress = false;
      refreshPromise = null;
    }
  })();

  // Wait for the refresh to complete
  return await refreshPromise;
}

// Main getAuth function using both caching and mutex
export const getAuth = cache(async (_req?: Request): Promise<GetAuthResult> => {
  const VERCEL_URL = env().VERCEL_URL;
  const baseUrl = VERCEL_URL || "http://localhost:3000/";

  try {
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
    const refreshToken = await getCookie(UNKEY_REFRESH_TOKEN);

    if (!sessionToken || !refreshToken) {
      return { userId: null, orgId: null, role: null };
    }

    // Use cached validation
    const validationResult = await validateSessionCached(sessionToken);

    // If session is valid, return user data
    if (validationResult.isValid && validationResult.userId) {
      return {
        userId: validationResult.userId,
        orgId: validationResult.orgId ?? null,
        role: validationResult.role ?? null,
      };
    }

    // If refresh is needed, use mutex-protected refresh
    if (validationResult.shouldRefresh) {
      const refreshResult = await refreshTokenWithMutex(refreshToken, baseUrl);

      // // If refresh returned no user, redirect to sign-in
      // if (!refreshResult.userId) {
      //   return {
      //     userId: null,
      //     orgId: null,
      //     role: null,
      //   };
      // }

      return refreshResult;
    }

    // Session is invalid and refresh not recommended
    return { userId: null, orgId: null, role: null };
  } catch (error) {
    console.error("Auth error:", error);
    return { userId: null, orgId: null, role: null };
  }
});
