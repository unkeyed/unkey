import { getAuthCookieOptions } from "@/lib/auth/cookie-security";
import { setCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import { UNKEY_SESSION_COOKIE } from "@/lib/auth/types";

export async function POST(request: Request) {
  try {
    // Get the current token from the request
    const currentToken = request.headers.get("x-current-token");
    if (!currentToken) {
      return Response.json({ success: false, error: "Failed to refresh session" }, { status: 401 });
    }
    // Call refreshSession logic here and get new token
    const { newToken, expiresAt } = await auth.refreshSession(currentToken);

    // Set the new cookie
    await setCookie({
      name: UNKEY_SESSION_COOKIE,
      value: newToken,
      options: {
        ...getAuthCookieOptions(),
        maxAge: Math.floor((expiresAt.getTime() - Date.now()) / 1000), // seconds
      },
    });

    return Response.json({ success: true });
  } catch (error) {
    console.error("Session refresh failed:", error);
    return Response.json({ success: false, error: "Failed to refresh session" }, { status: 401 });
  }
}
