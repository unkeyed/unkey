import { isSafeRedirectPath } from "@/app/auth/sign-in/redirect-utils";
import { setCookiesOnResponse } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import { AuthErrorCode, SIGN_IN_URL } from "@/lib/auth/types";
import { type NextRequest, NextResponse } from "next/server";
export async function GET(request: NextRequest) {
  const authResult = await auth.completeOAuthSignIn(request);

  if (!authResult.success) {
    if (
      (authResult.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED ||
        authResult.code === AuthErrorCode.EMAIL_VERIFICATION_REQUIRED) &&
      authResult.cookies &&
      authResult.cookies?.length > 0 // make typescript happy
    ) {
      const url = new URL(SIGN_IN_URL, request.url);

      // Preserve the redirect URL from OAuth state for deep link support
      const state = request.nextUrl.searchParams.get("state");
      if (state) {
        try {
          const parsed: unknown = JSON.parse(decodeURIComponent(state));
          const redirectUrlComplete =
            typeof parsed === "object" &&
            parsed !== null &&
            "redirectUrlComplete" in parsed &&
            typeof (parsed as { redirectUrlComplete: unknown }).redirectUrlComplete === "string"
              ? (parsed as { redirectUrlComplete: string }).redirectUrlComplete
              : null;
          if (
            redirectUrlComplete &&
            redirectUrlComplete !== "/apis" &&
            isSafeRedirectPath(redirectUrlComplete)
          ) {
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

      const response = NextResponse.redirect(url);

      return await setCookiesOnResponse(response, authResult.cookies);
    }

    // Handle other errors
    return NextResponse.redirect(new URL(SIGN_IN_URL, request.url));
  }

  // Get base URL from request because Next.js wants it
  const baseUrl = new URL(request.url).origin;
  const response = NextResponse.redirect(new URL(authResult.redirectTo, baseUrl));

  return await setCookiesOnResponse(response, authResult.cookies);
}
