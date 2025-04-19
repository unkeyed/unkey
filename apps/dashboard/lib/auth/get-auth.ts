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

// Per-user mutex tracking for refresh operations
// maps user tokens to their refresh operation promises
const refreshOperations = new Map<string, Promise<GetAuthResult>>();

// Refresh token with per-user mutex protection
async function refreshTokenWithUserMutex(
  refreshToken: string,
  baseUrl: string,
): Promise<GetAuthResult> {
  // If refresh is already in progress for this specific user, wait for it
  if (refreshOperations.has(refreshToken)) {
    try {
      return await refreshOperations.get(refreshToken)!;
    } catch (error) {
      console.error("Error while waiting for refresh:", error);
      // fall-through to continue with a new refresh attempt if the existing one fails
    }
  }

  // Create a refresh promise for this specific user
  const refreshPromise = (async (): Promise<GetAuthResult> => {
    try {
      const refreshResult = await fetch(`${baseUrl}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: {
          "x-refresh-token": refreshToken,
        },
      });

      if (!refreshResult.ok) {
        throw new Error(`Refresh failed: ${refreshResult.status}`);
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
      // only remove this user's refresh operation when done
      refreshOperations.delete(refreshToken);
    }
  })();

  // store this user's refresh promise in the map
  refreshOperations.set(refreshToken, refreshPromise);

  // wait for the refresh to complete
  return await refreshPromise;
}

// main getAuth function using both caching and per-user mutex
export const getAuth = cache(async (_req?: Request): Promise<GetAuthResult> => {
  const VERCEL_URL = env().VERCEL_URL;
  const baseUrl = VERCEL_URL || "http://localhost:3000";

  try {
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
    const refreshToken = await getCookie(UNKEY_REFRESH_TOKEN);

    if (!sessionToken || !refreshToken) {
      return { userId: null, orgId: null, role: null };
    }

    const validationResult = await validateSessionCached(sessionToken);

    // If session is valid, return user data
    if (validationResult.isValid && validationResult.userId) {
      return {
        userId: validationResult.userId,
        orgId: validationResult.orgId ?? null,
        role: validationResult.role ?? null,
      };
    }

    // If refresh is needed, use per-user mutex-protected refresh
    if (validationResult.shouldRefresh) {
      const refreshResult = await refreshTokenWithUserMutex(refreshToken, baseUrl);

      // security check: if we know the expected userId,
      // verify the refresh returned the correct user
      if (
        validationResult.userId &&
        refreshResult.userId &&
        refreshResult.userId !== validationResult.userId
      ) {
        console.error("User ID mismatch after refresh");
        return { userId: null, orgId: null, role: null };
      }

      return refreshResult;
    }

    // Session is invalid and refresh not recommended
    return { userId: null, orgId: null, role: null };
  } catch (error) {
    console.error("Auth error:", error);
    return { userId: null, orgId: null, role: null };
  }
});

// Export a function to clear a specific user's refresh operation
// useful to forcibly clear a stuck refresh operation
export function clearUserRefreshOperation(refreshToken: string): boolean {
  const hadOperation = refreshOperations.has(refreshToken);
  refreshOperations.delete(refreshToken);
  return hadOperation;
}

// for debugging: get count of ongoing refresh operations
export function getOngoingRefreshCount(): number {
  return refreshOperations.size;
}
