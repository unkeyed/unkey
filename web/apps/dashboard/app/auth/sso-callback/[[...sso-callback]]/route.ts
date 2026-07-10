import { setCookiesOnResponse } from "@/lib/auth/cookies";
import { sanitizeRedirectPath } from "@/lib/auth/redirect-utils";
import { auth } from "@/lib/auth/server";
import { AuthErrorCode, SIGN_IN_URL } from "@/lib/auth/types";
import { type NextRequest, NextResponse } from "next/server";
export async function GET(request: NextRequest) {
  const authResult = await auth.completeOAuthSignIn(request);

  if (!authResult.success) {
    if (
      (authResult.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED ||
        authResult.code === AuthErrorCode.EMAIL_VERIFICATION_REQUIRED ||
        "challengeType" in authResult) &&
      authResult.cookies &&
      authResult.cookies?.length > 0 // make typescript happy
    ) {
      // Org selection goes through /auth/continue, which auto-selects the
      // last used organization server-side before falling back to the
      // manual selector.
      const url = new URL(
        authResult.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED
          ? "/auth/continue"
          : SIGN_IN_URL,
        request.url,
      );

      // Preserve the redirect URL from OAuth state for deep link support
      const state = request.nextUrl.searchParams.get("state");
      if (state) {
        try {
          const parsed: unknown = JSON.parse(decodeURIComponent(state));
          // An empty fallback means "no deep link": anything non-string or
          // unsafe is dropped, and the default destination needs no param.
          const redirectUrlComplete = sanitizeRedirectPath(
            parsed !== null && typeof parsed === "object"
              ? (parsed as Record<string, unknown>).redirectUrlComplete
              : null,
            "",
          );
          if (redirectUrlComplete && redirectUrlComplete !== "/apis") {
            url.searchParams.set("redirect", redirectUrlComplete);
          }
        } catch {
          // Ignore state parsing errors
        }
      }

      // Add orgs to searchParams to make it accessible to the client
      if ("organizations" in authResult) {
        url.searchParams.set("orgs", JSON.stringify(authResult.organizations));
      }

      // Add verify=email to searchParams to render the email verification component
      if (authResult.code === AuthErrorCode.EMAIL_VERIFICATION_REQUIRED) {
        url.searchParams.set("verify", "email");
      }

      // Render the matching MFA/Radar challenge component; the data needed to
      // complete the challenge travels in the HttpOnly cookies set below.
      if ("challengeType" in authResult) {
        url.searchParams.set("challenge", authResult.challengeType);
      }

      const response = NextResponse.redirect(url);

      return await setCookiesOnResponse(response, authResult.cookies);
    }

    // Surface the failure (e.g. a Radar block) on the sign-in page instead of
    // bouncing to a bare form with no explanation.
    const errorUrl = new URL(SIGN_IN_URL, request.url);
    errorUrl.searchParams.set("error", authResult.code);
    return NextResponse.redirect(errorUrl);
  }

  // Get base URL from request because Next.js wants it
  const baseUrl = new URL(request.url).origin;
  // The auth provider is the authoritative sanitizer of redirectTo (it
  // originates from the attacker-controllable OAuth `state` param); this
  // re-check is defense in depth at the redirect sink, because
  // `new URL("https://evil.com", baseUrl)` discards the base and would make
  // any absolute value an open redirect.
  const response = NextResponse.redirect(
    new URL(sanitizeRedirectPath(authResult.redirectTo), baseUrl),
  );

  return await setCookiesOnResponse(response, authResult.cookies);
}
