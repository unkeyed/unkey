import { processPostAuthInvitation } from "@/lib/auth";
import { getAuth } from "@/lib/auth/get-auth";
import { type NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  try {
    // Perform non-redirecting auth check. Deliberately request-less: with a
    // request object, updateSession puts a refreshed session cookie into a
    // Headers object this route never returns, consuming the single-use
    // refresh token while the response and later cookies() reads keep the
    // stale session. The request-less path writes via cookies().set, which
    // updates both the outgoing response and subsequent reads in this request.
    const { userId } = await getAuth();

    if (!userId) {
      return NextResponse.json(
        { success: false, error: "User not authenticated" },
        { status: 401 },
      );
    }

    // Get the invitation token from the request body
    let body: { invitationToken?: unknown };
    try {
      body = await request.json();
    } catch (_error) {
      return NextResponse.json({ success: false, error: "Invalid JSON body" }, { status: 400 });
    }
    const token = typeof body?.invitationToken === "string" ? body.invitationToken.trim() : "";
    if (!token) {
      return NextResponse.json(
        { success: false, error: "Invitation token is required" },
        { status: 400 },
      );
    }

    // Process the invitation
    const result = await processPostAuthInvitation(token, userId);

    if (!result.success) {
      // processPostAuthInvitation returns only fixed, user-safe literals
      // (never raw provider errors), so they can be surfaced directly.
      return NextResponse.json({ success: false, error: result.error }, { status: 400 });
    }

    return NextResponse.json({
      success: true,
      organizationId: result.organizationId,
      switched: result.switched,
    });
  } catch (_error) {
    return NextResponse.json(
      { success: false, error: "Internal server error" },
      { status: 500, headers: { "Cache-Control": "no-store" } },
    );
  }
}
