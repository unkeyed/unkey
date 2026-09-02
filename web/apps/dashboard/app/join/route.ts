import { type NextRequest, NextResponse } from "next/server";
import { normalizeEmail, processPostAuthInvitation } from "@/lib/auth";
import { getAuth } from "@/lib/auth/get-auth";
import { auth } from "@/lib/auth/server";
import type { Invitation, User } from "@/lib/auth/types";

export const dynamic = "force-dynamic";
export async function GET(request: NextRequest) {
  const DASHBOARD_URL = new URL("/apis", request.url);
  const SIGN_IN_URL = new URL("/auth/sign-in", request.url);
  const SIGN_UP_URL = new URL("/auth/sign-up", request.url);

  const searchParams = request.nextUrl.searchParams;
  const invitationToken = searchParams.get("invitation_token");

  if (!invitationToken) {
    return NextResponse.redirect(DASHBOARD_URL);
  }

  // Request-less on purpose. The request-bound path refreshes the session into
  // a Headers object this handler never returns, spending the single-use
  // refresh token while auth.switchOrg (which reads the cookie store without a
  // request) would still see the stale session. getAuth never throws; it
  // reports an unusable session as a null userId.
  const { userId } = await getAuth();

  if (userId) {
    // Both invitation entry points run the same accept-then-switch
    // implementation, which fetches and validates the invitation itself.
    const result = await processPostAuthInvitation(invitationToken, userId);

    if (!result.success) {
      // Forward the token to the dashboard so PostAuthInvitationHandler can
      // retry and surface a message. Redirecting to a token-less /apis would
      // silently drop the user with no workspace and no explanation, the exact
      // ENG-3014 symptom this flow fixes on the client side.
      const RETRY_URL = new URL("/apis", request.url);
      RETRY_URL.searchParams.set("invitation_token", invitationToken);
      return NextResponse.redirect(RETRY_URL);
    }

    const JOIN_SUCCESS_URL = new URL("/join/success", request.url);
    JOIN_SUCCESS_URL.searchParams.set("from_invite", "true");
    // The user joined the org but we could not make it active for this
    // session, so the success page tells them to pick it from the switcher
    // rather than promising a workspace they will not land in.
    if (!result.switched) {
      JOIN_SUCCESS_URL.searchParams.set("switch_required", "true");
    }

    try {
      const org = await auth.getOrg(result.organizationId);
      if (org?.name) {
        JOIN_SUCCESS_URL.searchParams.set("org_name", org.name);
      }
    } catch (error) {
      // The org name is cosmetic. Never fail a completed join over it.
      console.warn("Could not fetch organization name for success page:", error);
    }

    return NextResponse.redirect(JOIN_SUCCESS_URL);
  }

  // Unauthenticated: the invitation only decides which auth page to send them
  // to, and is accepted later by the post-auth handler once they have a
  // session. It is never consumed here.
  let invitation: Invitation | null = null;
  try {
    invitation = await auth.getInvitation(invitationToken);
  } catch (error) {
    console.error("Failed to retrieve invitation:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return NextResponse.redirect(DASHBOARD_URL);
  }

  if (invitation?.state !== "pending") {
    return NextResponse.redirect(DASHBOARD_URL);
  }

  // The invited address is what the account must be looked up and created
  // under, so normalize it here too. A padded or mixed-case invite would
  // otherwise miss the existing account and push the user into sign-up.
  const normalizedEmail = normalizeEmail(invitation.email);

  let existingUser: User | null = null;
  try {
    existingUser = await auth.findUser(normalizedEmail);
  } catch (error) {
    console.error("Error checking for existing user:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    // Default to the sign-up flow if we cannot check.
  }

  const AUTH_URL = existingUser ? SIGN_IN_URL : SIGN_UP_URL;
  AUTH_URL.searchParams.set("invitation_token", invitationToken);
  AUTH_URL.searchParams.set("email", normalizedEmail);
  return NextResponse.redirect(AUTH_URL);
}
