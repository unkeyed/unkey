import { getCurrentUser } from "@/lib/auth";
import { setSessionCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import { updateSession } from "@/lib/auth/sessions";
import type { AuthenticatedUser, Invitation, User } from "@/lib/auth/types";
import { type NextRequest, NextResponse } from "next/server";
export async function GET(request: NextRequest) {
  const DASHBOARD_URL = new URL("/apis", request.url);
  const SIGN_IN_URL = new URL("/auth/sign-in", request.url);
  const SIGN_UP_URL = new URL("/auth/sign-up", request.url);

  const searchParams = request.nextUrl.searchParams;
  const invitationToken = searchParams.get("invitation_token");

  if (!invitationToken) {
    return NextResponse.redirect(DASHBOARD_URL); // middleware will pickup if they are not authenticated and redirect to login
  }

  // Check authentication status more reliably by validating the session
  let user: AuthenticatedUser | null = null;
  let isAuthenticated = false;

  try {
    // First validate the session to ensure we have a valid auth state
    const { session } = await updateSession(request);

    if (session?.userId) {
      user = await getCurrentUser();
      isAuthenticated = true;
    }
  } catch (_error) {
    // User is not authenticated, which is fine - we'll handle this below
  }

  // exchange token for invitation
  let invitation: Invitation | null = null;
  try {
    invitation = await auth.getInvitation(invitationToken);
  } catch (error) {
    console.error("Failed to retrieve invitation:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return NextResponse.redirect(DASHBOARD_URL);
  }

  if (!invitation) {
    return NextResponse.redirect(DASHBOARD_URL);
  }

  const { email: invitationEmail, state, organizationId, id: invitationId } = invitation;

  if (state !== "pending") {
    // Add invitation_token to the dashboard URL so our post-auth handler can still try to process it
    DASHBOARD_URL.searchParams.set("invitation_token", invitationToken);
    return NextResponse.redirect(DASHBOARD_URL);
  }

  // if they are authenticated
  if (user && isAuthenticated) {
    if (user.email !== invitationEmail) {
      return NextResponse.redirect(DASHBOARD_URL);
    }

    if (!organizationId) {
      return NextResponse.redirect(DASHBOARD_URL);
    }

    try {
      // Accept invitation first
      await auth.acceptInvitation(invitationId);

      // Switch organization and get the new session token
      const { newToken, expiresAt } = await auth.switchOrg(organizationId);

      if (!newToken || !expiresAt) {
        throw new Error("Invalid session data returned from auth provider");
      }

      // Set the session cookie securely on the server side
      await setSessionCookie({ token: newToken, expiresAt });

      // Redirect to success page with organization context
      const JOIN_SUCCESS_URL = new URL("/join/success", request.url);
      JOIN_SUCCESS_URL.searchParams.set("from_invite", "true");

      // Try to get organization name for better UX
      try {
        const org = await auth.getOrg(organizationId);
        if (org?.name) {
          JOIN_SUCCESS_URL.searchParams.set("org_name", org.name);
        }
      } catch (error) {
        // Don't fail the redirect if we can't get org name
        console.warn("Could not fetch organization name for success page:", error);
      }

      return NextResponse.redirect(JOIN_SUCCESS_URL);
    } catch (error) {
      console.error("Failed to accept invitation:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      // Add invitation_token to dashboard URL so post-auth handler can retry
      DASHBOARD_URL.searchParams.set("invitation_token", invitationToken);
      return NextResponse.redirect(DASHBOARD_URL);
    }
  }

  let existingUser: User | null = null;
  try {
    existingUser = await auth.findUser(invitationEmail);
  } catch (error) {
    console.error("Error checking for existing user:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    // Default to sign-up flow if we can't check
  }

  if (existingUser) {
    SIGN_IN_URL.searchParams.set("invitation_token", invitationToken);
    SIGN_IN_URL.searchParams.set("email", invitationEmail);
    return NextResponse.redirect(SIGN_IN_URL);
  }
  SIGN_UP_URL.searchParams.set("invitation_token", invitationToken);
  SIGN_UP_URL.searchParams.set("email", invitationEmail);
  return NextResponse.redirect(SIGN_UP_URL);
}
