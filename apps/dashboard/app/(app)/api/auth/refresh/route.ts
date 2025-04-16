import { getCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import {
  UNKEY_ACCESS_MAX_AGE,
  UNKEY_ACCESS_TOKEN,
  UNKEY_REFRESH_TOKEN,
  UNKEY_SESSION_COOKIE,
} from "@/lib/auth/types";

// Per-user mutex tracking for refresh operations
// maps refresh tokens to their refresh operation promises
const refreshOperations = new Map<string, Promise<Response>>();

export async function POST(request: Request) {
  try {
    // Get refresh token from header or cookie
    const currentRefreshToken =
      request.headers.get("x-refresh-token") || (await getCookie(UNKEY_REFRESH_TOKEN));

    if (!currentRefreshToken) {
      console.error("Access token refresh failed: no refresh token");
      return Response.json(
        { success: false, error: "Failed to refresh access token" },
        { status: 401 },
      );
    }

    // Handle concurrent refresh attempts with per-user mutex pattern
    if (refreshOperations.has(currentRefreshToken)) {
      try {
        // Wait for existing refresh to complete for this specific token
        return await refreshOperations.get(currentRefreshToken)!;
      } catch (error) {
        console.error("Error while waiting for refresh:", error);
        // fall-through to continue with a new refresh attempt
      }
    }

    // Create refresh promise for this specific token
    const refreshPromise = (async () => {
      try {
        // Call refreshAccessToken to get new tokens
        const { sessionToken, accessToken, refreshToken, expiresAt, session } =
          await auth.refreshAccessToken(currentRefreshToken);

        // Calculate max age
        const sessionMaxAge = Math.floor((expiresAt.getTime() - Date.now()) / 1000); // 7 days in seconds

        // Create response with session data
        const response = Response.json({ success: true, session });

        // Set all cookies on the response
        response.headers.append(
          "Set-Cookie",
          `${UNKEY_SESSION_COOKIE}=${sessionToken}; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=${sessionMaxAge}`,
        );

        response.headers.append(
          "Set-Cookie",
          `${UNKEY_ACCESS_TOKEN}=${accessToken}; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=${UNKEY_ACCESS_MAX_AGE}`,
        );

        response.headers.append(
          "Set-Cookie",
          `${UNKEY_REFRESH_TOKEN}=${refreshToken}; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=${sessionMaxAge}`,
        );

        return response;
      } catch (error) {
        console.error("Refresh failed:", error);
        return Response.json(
          { success: false, error: "Failed to refresh access token" },
          { status: 401 },
        );
      } finally {
        // Clean up this token's refresh operation when done
        refreshOperations.delete(currentRefreshToken);
      }
    })();

    // Store this token's refresh promise in the map
    refreshOperations.set(currentRefreshToken, refreshPromise);

    // Return the promise result
    return await refreshPromise;
  } catch (error) {
    console.error("Unexpected error in refresh:", error);
    return Response.json({ success: false, error: "Failed to refresh session" }, { status: 500 });
  }
}
