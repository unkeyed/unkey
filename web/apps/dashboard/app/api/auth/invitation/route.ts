import { processPostAuthInvitation } from "@/lib/auth";
import { getAuth } from "@/lib/auth/get-auth";
import { auth } from "@/lib/auth/server";
import * as Sentry from "@sentry/nextjs";
import { type NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  try {
    // Perform non-redirecting auth check
    const { userId } = await getAuth(request);

    if (!userId) {
      return NextResponse.json(
        { success: false, error: "User not authenticated" },
        { status: 401 },
      );
    }

    // Get the user data
    const user = await auth.getUser(userId);
    if (!user) {
      return NextResponse.json(
        { success: false, error: "User not authenticated" },
        { status: 401 },
      );
    }

    // Get the invitation token from the request body
    const body = await request.json();
    const token = typeof body?.invitationToken === "string" ? body.invitationToken.trim() : "";
    if (!token) {
      return NextResponse.json(
        { success: false, error: "Invitation token is required" },
        { status: 400 },
      );
    }

    // Process the invitation
    const result = await processPostAuthInvitation(token, user.id);

    if (!result.success) {
      return NextResponse.json(
        { success: false, error: "Invalid or expired invitation" },
        { status: 400 },
      );
    }

    return NextResponse.json({
      success: true,
      organizationId: result.organizationId,
    });
  } catch (error) {
    Sentry.captureException(error, { tags: { handler: "auth_invitation" } });
    return NextResponse.json(
      { success: false, error: "Internal server error" },
      { status: 500, headers: { "Cache-Control": "no-store" } },
    );
  }
}
