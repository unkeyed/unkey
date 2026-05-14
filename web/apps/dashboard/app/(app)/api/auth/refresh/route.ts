import { getAuthCookieOptions } from "@/lib/auth/cookie-security";
import { setCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import { UNKEY_SESSION_COOKIE } from "@/lib/auth/types";
import * as Sentry from "@sentry/nextjs";

export async function POST(request: Request) {
  try {
    const currentToken = request.headers.get("x-current-token");
    if (!currentToken) {
      return Response.json({ success: false, error: "Failed to refresh session" }, { status: 401 });
    }
    const { newToken, expiresAt } = await auth.refreshSession(currentToken);

    await setCookie({
      name: UNKEY_SESSION_COOKIE,
      value: newToken,
      options: {
        ...getAuthCookieOptions(),
        maxAge: Math.floor((expiresAt.getTime() - Date.now()) / 1000),
      },
    });

    return Response.json({ success: true });
  } catch (error) {
    // Session refresh failures aren't always actionable (expired tokens are normal),
    // but persistent failures point to provider outages or token-format bugs we
    // *do* want visibility on — so report and let Sentry's grouping handle volume.
    Sentry.captureException(error, { tags: { handler: "auth_refresh" } });
    console.error("Session refresh failed:", error);
    return Response.json({ success: false, error: "Failed to refresh session" }, { status: 401 });
  }
}
