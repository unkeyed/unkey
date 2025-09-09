import { acceptInvitationAndSwitchOrg, getCurrentUser } from "@/lib/auth";
import { auth } from "@/lib/auth/server";
import { updateSession } from "@/lib/auth/sessions";
import type { Invitation } from "@/lib/auth/types";
import { type NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  try {
    // Validate session first
    const { session } = await updateSession(request);
    if (!session || !session.userId) {
      return NextResponse.json(
        { success: false, error: "User not authenticated" },
        { status: 401 },
      );
    }

    // Get the current authenticated user
    const user = await getCurrentUser();
    if (!user) {
      return NextResponse.json({ success: false, error: "User not found" }, { status: 401 });
    }

    // Get the invitation details from the request body
    const body = await request.json();
    const { invitationToken } = body;

    if (!invitationToken) {
      return NextResponse.json(
        { success: false, error: "Invitation token is required" },
        { status: 400 },
      );
    }

    // Get invitation details
    let invitation: Invitation | null;
    try {
      invitation = await auth.getInvitation(invitationToken);
    } catch (error) {
      console.error("Failed to retrieve invitation:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return NextResponse.json(
        { success: false, error: "Invalid or expired invitation token" },
        { status: 400 },
      );
    }

    if (!invitation) {
      return NextResponse.json({ success: false, error: "Invitation not found" }, { status: 404 });
    }

    const { email: invitationEmail, state, organizationId, id: invitationId } = invitation;

    // Validate invitation state
    if (state !== "pending") {
      return NextResponse.json(
        { success: false, error: `Invitation is ${state}` },
        { status: 400 },
      );
    }

    // Validate email matches
    if (user.email !== invitationEmail) {
      return NextResponse.json({ success: false, error: "Email mismatch" }, { status: 403 });
    }

    if (!organizationId) {
      return NextResponse.json(
        { success: false, error: "No organization ID in invitation" },
        { status: 400 },
      );
    }

    // Accept invitation and switch organization
    try {
      await acceptInvitationAndSwitchOrg(invitationId, organizationId);

      // Create response with updated session headers
      const response = NextResponse.json({
        success: true,
        organizationId,
        message: "Invitation accepted and organization switched successfully",
      });

      // Update session to ensure new session state is reflected
      const { headers: sessionHeaders } = await updateSession(request);

      // Copy session headers to response
      for (const [key, value] of sessionHeaders.entries()) {
        if (key.toLowerCase().includes("cookie") || key.toLowerCase().includes("session")) {
          response.headers.set(key, value);
        }
      }

      return response;
    } catch (error) {
      console.error("Failed to accept invitation:", {
        error: error instanceof Error ? error.message : "Unknown error",
      });
      return NextResponse.json(
        {
          success: false,
          error: error instanceof Error ? error.message : "Failed to accept invitation",
        },
        { status: 500 },
      );
    }
  } catch (error) {
    console.error("Error in accept invitation API:", error);
    return NextResponse.json({ success: false, error: "Internal server error" }, { status: 500 });
  }
}
