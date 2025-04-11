import { getCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import {
  UNKEY_ACCESS_MAX_AGE,
  UNKEY_ACCESS_TOKEN,
  UNKEY_REFRESH_TOKEN,
  UNKEY_SESSION_COOKIE,
} from "@/lib/auth/types";

// Global mutex for refresh operations
let refreshInProgress = false;
let refreshPromise: Promise<Response> | null = null;

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

    // Handle concurrent refresh attempts with mutex pattern
    if (refreshInProgress && refreshPromise) {
      try {
        // Wait for existing refresh to complete
        return await refreshPromise;
      } catch (error) {
        console.error("Error while waiting for refresh:", error);
        return Response.json(
          { success: false, error: "Failed to refresh access token" },
          { status: 401 },
        );
      }
    }

    // Set mutex to prevent concurrent refreshes
    refreshInProgress = true;

    // Create refresh promise
    refreshPromise = (async () => {
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
          `${UNKEY_SESSION_COOKIE}=${sessionToken}; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=${sessionMaxAge}`,
        );

        response.headers.append(
          "Set-Cookie",
          `${UNKEY_ACCESS_TOKEN}=${accessToken}; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=${UNKEY_ACCESS_MAX_AGE}`,
        );

        response.headers.append(
          "Set-Cookie",
          `${UNKEY_REFRESH_TOKEN}=${refreshToken}; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=${sessionMaxAge}`,
        );

        return response;
      } catch (error) {
        console.error("Refresh failed:", error);
        return Response.json(
          { success: false, error: "Failed to refresh access token" },
          { status: 401 },
        );
      } finally {
        // Clear mutex
        refreshInProgress = false;
        refreshPromise = null;
      }
    })();

    // Return the promise result
    return await refreshPromise;
  } catch (error) {
    console.error("Unexpected error in refresh:", error);
    return Response.json({ success: false, error: "Failed to refresh session" }, { status: 500 });
  }
}
