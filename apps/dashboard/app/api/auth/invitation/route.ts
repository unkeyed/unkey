import { getCurrentUser, processPostAuthInvitation } from "@/lib/auth";
import { type NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  try {
    // Get the current authenticated user
    const user = await getCurrentUser();

    if (!user) {
      return NextResponse.json(
        { success: false, error: "User not authenticated" },
        { status: 401 },
      );
    }

    // Get the invitation token from the request body
    const body = await request.json();
    const { invitationToken } = body;

    if (!invitationToken) {
      return NextResponse.json(
        { success: false, error: "Invitation token is required" },
        { status: 400 },
      );
    }

    // Process the invitation
    const result = await processPostAuthInvitation(invitationToken, user.id);

    if (!result.success) {
      return NextResponse.json({ success: false, error: result.error }, { status: 400 });
    }

    return NextResponse.json({
      success: true,
      organizationId: result.organizationId,
    });
  } catch (error) {
    console.error("Error processing invitation:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return NextResponse.json({ success: false, error: "Internal server error" }, { status: 500 });
  }
}
