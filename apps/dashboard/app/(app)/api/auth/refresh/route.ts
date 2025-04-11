import { getCookie, setCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import { tokenManager } from "@/lib/auth/token-management-service";
import {
  UNKEY_ACCESS_MAX_AGE,
  UNKEY_ACCESS_TOKEN,
  UNKEY_REFRESH_MAX_AGE,
  UNKEY_REFRESH_TOKEN,
  UNKEY_SESSION_COOKIE,
  UNKEY_USER_IDENTITY_COOKIE,
  UNKEY_USER_IDENTITY_MAX_AGE,
} from "@/lib/auth/types";
import { type NextRequest, NextResponse } from "next/server";

export async function POST(req: NextRequest) {
  try {
    // Get the refresh token from the cookie
    const refreshToken = await getCookie(UNKEY_REFRESH_TOKEN);

    // Get the user identity from the request headers and cookies
    const requestUserIdentity = req.headers.get("x-user-identity");
    const cookieUserIdentity = await getCookie(UNKEY_USER_IDENTITY_COOKIE);

    // If we don't have a refresh token, return error
    if (!refreshToken) {
      return NextResponse.json({ error: "No refresh token available" }, { status: 401 });
    }

    // Check user identity consistency
    // 1. If we have a user identity in the cookie, it should match the request
    // 2. If we don't have one in the cookie but have one in the request, accept it
    if (cookieUserIdentity && requestUserIdentity && cookieUserIdentity !== requestUserIdentity) {
      console.warn("User identity mismatch during refresh");
      return NextResponse.json({ error: "Invalid user identity" }, { status: 403 });
    }

    // Use either the cookie identity or request identity
    const userIdentity = cookieUserIdentity || requestUserIdentity;

    // If we have a user identity, verify token ownership
    if (userIdentity) {
      const isValidOwner = tokenManager.verifyTokenOwnership({
        refreshToken,
        userIdentity,
      });

      if (!isValidOwner) {
        console.warn("Refresh token ownership verification failed");
        return NextResponse.json({ error: "Invalid refresh token ownership" }, { status: 403 });
      }

      // If the user identity is only in the request, set it as a cookie
      if (!cookieUserIdentity && requestUserIdentity) {
        await setCookie({
          name: UNKEY_USER_IDENTITY_COOKIE,
          value: requestUserIdentity,
          options: {
            httpOnly: true,
            secure: true,
            sameSite: "strict",
            maxAge: UNKEY_USER_IDENTITY_MAX_AGE,
            path: "/",
          },
        });
      }
    }

    // continue with token refresh
    const result = await auth.refreshAccessToken(refreshToken);

    if (!result || !result.session) {
      // Remove the invalid token from our ownership mapping
      tokenManager.removeToken(refreshToken);

      return NextResponse.json({ error: "Failed to refresh session" }, { status: 401 });
    }

    // Calculate remaining time for session in seconds
    const sessionMaxAge = Math.floor((result.expiresAt.getTime() - Date.now()) / 1000);

    // Update session cookie
    await setCookie({
      name: UNKEY_SESSION_COOKIE,
      value: result.sessionToken,
      options: {
        httpOnly: true,
        secure: true,
        sameSite: "strict",
        maxAge: sessionMaxAge,
        path: "/",
      },
    });

    // Set access token cookie if available
    if (result.accessToken) {
      await setCookie({
        name: UNKEY_ACCESS_TOKEN,
        value: result.accessToken,
        options: {
          httpOnly: true,
          secure: true,
          sameSite: "strict",
          maxAge: UNKEY_ACCESS_MAX_AGE,
          path: "/",
        },
      });
    }

    // Update refresh token if available
    if (result.refreshToken && result.refreshToken !== refreshToken) {
      await setCookie({
        name: UNKEY_REFRESH_TOKEN,
        value: result.refreshToken,
        options: {
          httpOnly: true,
          secure: true,
          sameSite: "strict",
          maxAge: UNKEY_REFRESH_MAX_AGE,
          path: "/",
        },
      });

      // Update token ownership mapping
      if (userIdentity) {
        tokenManager.updateTokenOwnership({
          oldToken: refreshToken,
          newToken: result.refreshToken,
          userIdentity: userIdentity,
        });
      }
    }

    // Return success response with access token
    return NextResponse.json({
      success: true,
      userId: result.session.userId,
      orgId: result.session.orgId,
      role: result.session.role,
      accessToken: result.accessToken,
      expiresAt: result.expiresAt,
    });
  } catch (error) {
    console.error("Refresh error:", error);
    return NextResponse.json({ error: "Server error during refresh" }, { status: 500 });
  }
}
