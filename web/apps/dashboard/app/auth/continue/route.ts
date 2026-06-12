import { isSafeRedirectPath, resolveRedirectUrl } from "@/app/auth/sign-in/redirect-utils";
import { setCookiesOnResponse } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import { PENDING_SESSION_COOKIE, SIGN_IN_URL, UNKEY_LAST_ORG_COOKIE } from "@/lib/auth/types";
import { db } from "@/lib/db";
import { type NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

/**
 * Continues an authentication that ended in organization_selection_required.
 *
 * Auto-selects the last used organization entirely server-side, so users
 * with a remembered workspace go from the auth flow to the dashboard in one
 * redirect chain without any intermediate page painting (the previous
 * client-side version flashed a dark loading screen before the themed
 * dashboard). Users without a usable remembered org fall back to the manual
 * org selector on the sign-in page.
 */
export async function GET(request: NextRequest) {
  const orgsParam = request.nextUrl.searchParams.get("orgs");
  const redirectParam = request.nextUrl.searchParams.get("redirect");

  // Fallback destination: the manual org selector on the sign-in page
  const selectorUrl = new URL(SIGN_IN_URL, request.url);
  if (orgsParam) {
    selectorUrl.searchParams.set("orgs", orgsParam);
  }
  if (redirectParam) {
    selectorUrl.searchParams.set("redirect", redirectParam);
  }

  const pendingToken = request.cookies.get(PENDING_SESSION_COOKIE)?.value;
  if (!pendingToken) {
    return NextResponse.redirect(new URL(SIGN_IN_URL, request.url));
  }

  const lastUsedOrgId = request.cookies.get(UNKEY_LAST_ORG_COOKIE)?.value;
  if (!lastUsedOrgId) {
    return NextResponse.redirect(selectorUrl);
  }

  try {
    const result = await auth.completeOrgSelection({
      orgId: lastUsedOrgId,
      pendingAuthToken: pendingToken,
    });

    if (!result.success) {
      // Selecting the org can itself be interrupted (e.g. an MFA challenge);
      // hand off to the matching challenge UI with its cookies in place.
      if ("challengeType" in result && result.cookies) {
        const challengeUrl = new URL(SIGN_IN_URL, request.url);
        challengeUrl.searchParams.set("challenge", result.challengeType);
        if (redirectParam) {
          challengeUrl.searchParams.set("redirect", redirectParam);
        }
        const response = NextResponse.redirect(challengeUrl);
        return await setCookiesOnResponse(response, result.cookies);
      }

      // The remembered org is stale (revoked membership, expired token, ...).
      // Clear it so the selector doesn't auto-retry, and let the user pick.
      const response = NextResponse.redirect(selectorUrl);
      response.cookies.delete(UNKEY_LAST_ORG_COOKIE);
      return response;
    }

    // Rewrite a deep link to the selected workspace's slug when possible
    let destination = result.redirectTo;
    if (redirectParam && isSafeRedirectPath(redirectParam)) {
      let workspaceSlug: string | undefined;
      try {
        const workspace = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.orgId, lastUsedOrgId), isNull(table.deletedAtM)),
          columns: { slug: true },
        });
        workspaceSlug = workspace?.slug ?? undefined;
      } catch {
        // Non-critical: fall back to the deep link as-is
      }
      destination = resolveRedirectUrl(redirectParam, workspaceSlug) ?? destination;
    }

    const response = NextResponse.redirect(new URL(destination, request.url));
    await setCookiesOnResponse(response, result.cookies);
    response.cookies.delete(PENDING_SESSION_COOKIE);
    return response;
  } catch (_error) {
    const response = NextResponse.redirect(selectorUrl);
    response.cookies.delete(UNKEY_LAST_ORG_COOKIE);
    return response;
  }
}
