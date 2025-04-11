import { cache } from "react";
import { env } from "../env";
import { getCookie } from "./cookies";
import { auth } from "./server";
import { tokenManager } from "./token-management-service";
import {
  type AuthResult,
  type SessionValidationResult,
  UNKEY_ACCESS_MAX_AGE,
  UNKEY_ACCESS_TOKEN,
  UNKEY_REFRESH_TOKEN,
  UNKEY_SESSION_COOKIE,
  UNKEY_USER_IDENTITY_COOKIE,
} from "./types";

const CACHE_TTL = 10 * 1000;

/**
 * Simple cache for validation
 *
 * getAuth is called in the trpc context for every trpc route to ensure authentication.
 * Pages may have client components that call multiple trpc routers to fetch data,
 * resulting in validation being called multiple times.
 * This cache will store the validation value for a short time, e.g. 10 seconds,
 * just long enough for the concurrent trpc routes to use the same value before being invalidated.
 */
const sessionValidationCache = new Map<
  string,
  {
    result: {
      isValid: boolean;
      shouldRefresh: boolean;
      userId?: string;
      orgId?: string | null;
      role?: string | null;
      accessToken?: string;
      expiresAt?: Date;
    };
    expiresAt: number;
  }
>();

export async function validateSession(
  sessionToken: string | null,
): Promise<SessionValidationResult> {
  if (!sessionToken) {
    return { isValid: false, shouldRefresh: false };
  }
  try {
    return await auth.validateSession(sessionToken);
  } catch (error) {
    console.error("Session validation error:", error);
    return { isValid: false, shouldRefresh: false };
  }
}

// Per-user mutex tracking for refresh operations
// maps user tokens to their refresh operation promises
// TODO: move this to redis
const refreshOperations = new Map<string, Promise<AuthResult>>();

// Refresh token with per-user mutex protection, using the refresh token as the key
async function refreshTokenWithUserMutex(
  refreshToken: string,
  baseUrl: string,
  userIdentity: string | null = null,
): Promise<AuthResult> {
  // If we have a user identity, verify token ownership
  if (userIdentity) {
    const isValidOwner = tokenManager.verifyTokenOwnership({
      refreshToken,
      userIdentity,
    });

    if (!isValidOwner) {
      console.warn("Refresh token ownership verification failed");
      return { userId: null, orgId: null, role: null };
    }
  }

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
  const refreshPromise = (async (): Promise<AuthResult> => {
    try {
      // Prepare headers
      const headers: Record<string, string> = {};

      // Add the refresh token to headers
      headers["x-refresh-token"] = refreshToken;

      // Add user identity if available
      if (userIdentity) {
        headers["x-user-identity"] = userIdentity;
      }

      const refreshResult = await fetch(`${baseUrl}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers,
      });

      if (!refreshResult.ok) {
        throw new Error(`Refresh failed: ${refreshResult.status}`);
      }

      const refreshedData = await refreshResult.json();

      // If we received a new refresh token and have user identity, update mapping
      if (
        refreshedData.refreshToken &&
        refreshedData.refreshToken !== refreshToken &&
        userIdentity
      ) {
        tokenManager.updateTokenOwnership({
          oldToken: refreshToken,
          newToken: refreshedData.refreshToken,
          userIdentity,
        });
      }

      return {
        userId: refreshedData.userId || null,
        orgId: refreshedData.orgId || null,
        role: refreshedData.role || null,
        accessToken: refreshedData.accessToken || null,
        expiresAt: refreshedData.expiresAt || null,
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
export const getAuth = cache(async (_req?: Request): Promise<AuthResult> => {
  const VERCEL_URL = env().VERCEL_URL;
  const baseUrl = VERCEL_URL || "http://localhost:3000";

  try {
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
    const refreshToken = await getCookie(UNKEY_REFRESH_TOKEN);
    const accessToken = await getCookie(UNKEY_ACCESS_TOKEN);
    const userIdentity = await getCookie(UNKEY_USER_IDENTITY_COOKIE);

    if (!sessionToken || !refreshToken) {
      return { userId: null, orgId: null, role: null };
    }

    // Check if we have a valid access token first
    if (accessToken) {
      // Use cache to reduce validation calls
      const cacheKey = `${sessionToken}:${accessToken}:${refreshToken}`;
      const cachedValidation = sessionValidationCache.get(cacheKey);

      if (cachedValidation && cachedValidation.expiresAt > Date.now()) {
        const { result } = cachedValidation;

        // If cached result says session is valid, return the user data with access token
        if (result.isValid && result.userId) {
          return {
            userId: result.userId,
            orgId: result.orgId ?? null,
            role: result.role ?? null,
            accessToken: result.accessToken ?? null,
            expiresAt: result.expiresAt ?? null,
          };
        }

        // If cached result says refresh is needed, do that
        if (result.shouldRefresh) {
          return await refreshTokenWithUserMutex(refreshToken, baseUrl, userIdentity);
        }

        // Session invalid and refresh not recommended
        return { userId: null, orgId: null, role: null };
      }
    }

    // Validate the session
    const validationResult = await validateSession(sessionToken);

    // Only cache valid results that have an access token
    if (validationResult.isValid && validationResult.accessToken) {
      const cacheKey = `${sessionToken}:${validationResult.accessToken}:${refreshToken}`;
      sessionValidationCache.set(cacheKey, {
        result: validationResult,
        expiresAt: Date.now() + CACHE_TTL,
      });
    }

    // If session is valid, return user data with access token
    if (validationResult.isValid && validationResult.userId) {
      return {
        userId: validationResult.userId,
        orgId: validationResult.orgId ?? null,
        role: validationResult.role ?? null,
        accessToken: validationResult.accessToken ?? null,
        expiresAt: new Date(Date.now() + UNKEY_ACCESS_MAX_AGE),
      };
    }

    // If refresh is needed, use per-user mutex-protected refresh
    if (validationResult.shouldRefresh) {
      return await refreshTokenWithUserMutex(refreshToken, baseUrl, userIdentity);
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
